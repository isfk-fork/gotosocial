/*
   GoToSocial
   Copyright (C) 2021 GoToSocial Authors admin@gotosocial.org

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package message

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"
	apimodel "github.com/superseriousbusiness/gotosocial/internal/api/model"
	"github.com/superseriousbusiness/gotosocial/internal/config"
	"github.com/superseriousbusiness/gotosocial/internal/db"
	"github.com/superseriousbusiness/gotosocial/internal/federation"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
	"github.com/superseriousbusiness/gotosocial/internal/media"
	"github.com/superseriousbusiness/gotosocial/internal/oauth"
	"github.com/superseriousbusiness/gotosocial/internal/storage"
	"github.com/superseriousbusiness/gotosocial/internal/typeutils"
)

// Processor should be passed to api modules (see internal/apimodule/...). It is used for
// passing messages back and forth from the client API and the federating interface, via channels.
// It also contains logic for filtering which messages should end up where.
// It is designed to be used asynchronously: the client API and the federating API should just be able to
// fire messages into the processor and not wait for a reply before proceeding with other work. This allows
// for clean distribution of messages without slowing down the client API and harming the user experience.
type Processor interface {
	// ToClientAPI returns a channel for putting in messages that need to go to the gts client API.
	// ToClientAPI() chan gtsmodel.ToClientAPI
	// FromClientAPI returns a channel for putting messages in that come from the client api going to the processor
	FromClientAPI() chan gtsmodel.FromClientAPI
	// ToFederator returns a channel for putting in messages that need to go to the federator (activitypub).
	// ToFederator() chan gtsmodel.ToFederator
	// FromFederator returns a channel for putting messages in that come from the federator (activitypub) going into the processor
	FromFederator() chan gtsmodel.FromFederator
	// Start starts the Processor, reading from its channels and passing messages back and forth.
	Start() error
	// Stop stops the processor cleanly, finishing handling any remaining messages before closing down.
	Stop() error

	/*
		CLIENT API-FACING PROCESSING FUNCTIONS
		These functions are intended to be called when the API client needs an immediate (ie., synchronous) reply
		to an HTTP request. As such, they will only do the bare-minimum of work necessary to give a properly
		formed reply. For more intensive (and time-consuming) calls, where you don't require an immediate
		response, pass work to the processor using a channel instead.
	*/

	// AccountCreate processes the given form for creating a new account, returning an oauth token for that account if successful.
	AccountCreate(authed *oauth.Auth, form *apimodel.AccountCreateRequest) (*apimodel.Token, error)
	// AccountGet processes the given request for account information.
	AccountGet(authed *oauth.Auth, targetAccountID string) (*apimodel.Account, error)
	// AccountUpdate processes the update of an account with the given form
	AccountUpdate(authed *oauth.Auth, form *apimodel.UpdateCredentialsRequest) (*apimodel.Account, error)
	// AccountStatusesGet fetches a number of statuses (in time descending order) from the given account, filtered by visibility for
	// the account given in authed.
	AccountStatusesGet(authed *oauth.Auth, targetAccountID string, limit int, excludeReplies bool, maxID string, pinned bool, mediaOnly bool) ([]apimodel.Status, ErrorWithCode)
	// AccountFollowersGet fetches a list of the target account's followers.
	AccountFollowersGet(authed *oauth.Auth, targetAccountID string) ([]apimodel.Account, ErrorWithCode)
	// AccountFollowingGet fetches a list of the accounts that target account is following.
	AccountFollowingGet(authed *oauth.Auth, targetAccountID string) ([]apimodel.Account, ErrorWithCode)
	// AccountRelationshipGet returns a relationship model describing the relationship of the targetAccount to the Authed account.
	AccountRelationshipGet(authed *oauth.Auth, targetAccountID string) (*apimodel.Relationship, ErrorWithCode)
	// AccountFollowCreate handles a follow request to an account, either remote or local.
	AccountFollowCreate(authed *oauth.Auth, form *apimodel.AccountFollowRequest) (*apimodel.Relationship, ErrorWithCode)
	// AccountFollowRemove handles the removal of a follow/follow request to an account, either remote or local.
	AccountFollowRemove(authed *oauth.Auth, targetAccountID string) (*apimodel.Relationship, ErrorWithCode)

	// AdminEmojiCreate handles the creation of a new instance emoji by an admin, using the given form.
	AdminEmojiCreate(authed *oauth.Auth, form *apimodel.EmojiCreateRequest) (*apimodel.Emoji, error)

	// AppCreate processes the creation of a new API application
	AppCreate(authed *oauth.Auth, form *apimodel.ApplicationCreateRequest) (*apimodel.Application, error)

	// FileGet handles the fetching of a media attachment file via the fileserver.
	FileGet(authed *oauth.Auth, form *apimodel.GetContentRequestForm) (*apimodel.Content, error)

	// FollowRequestsGet handles the getting of the authed account's incoming follow requests
	FollowRequestsGet(auth *oauth.Auth) ([]apimodel.Account, ErrorWithCode)
	// FollowRequestAccept handles the acceptance of a follow request from the given account ID
	FollowRequestAccept(auth *oauth.Auth, accountID string) (*apimodel.Relationship, ErrorWithCode)

	// InstanceGet retrieves instance information for serving at api/v1/instance
	InstanceGet(domain string) (*apimodel.Instance, ErrorWithCode)

	// MediaCreate handles the creation of a media attachment, using the given form.
	MediaCreate(authed *oauth.Auth, form *apimodel.AttachmentRequest) (*apimodel.Attachment, error)
	// MediaGet handles the GET of a media attachment with the given ID
	MediaGet(authed *oauth.Auth, attachmentID string) (*apimodel.Attachment, ErrorWithCode)
	// MediaUpdate handles the PUT of a media attachment with the given ID and form
	MediaUpdate(authed *oauth.Auth, attachmentID string, form *apimodel.AttachmentUpdateRequest) (*apimodel.Attachment, ErrorWithCode)

	// StatusCreate processes the given form to create a new status, returning the api model representation of that status if it's OK.
	StatusCreate(authed *oauth.Auth, form *apimodel.AdvancedStatusCreateForm) (*apimodel.Status, error)
	// StatusDelete processes the delete of a given status, returning the deleted status if the delete goes through.
	StatusDelete(authed *oauth.Auth, targetStatusID string) (*apimodel.Status, error)
	// StatusFave processes the faving of a given status, returning the updated status if the fave goes through.
	StatusFave(authed *oauth.Auth, targetStatusID string) (*apimodel.Status, error)
	// StatusBoost processes the boost/reblog of a given status, returning the newly-created boost if all is well.
	StatusBoost(authed *oauth.Auth, targetStatusID string) (*apimodel.Status, ErrorWithCode)
	// StatusFavedBy returns a slice of accounts that have liked the given status, filtered according to privacy settings.
	StatusFavedBy(authed *oauth.Auth, targetStatusID string) ([]*apimodel.Account, error)
	// StatusGet gets the given status, taking account of privacy settings and blocks etc.
	StatusGet(authed *oauth.Auth, targetStatusID string) (*apimodel.Status, error)
	// StatusUnfave processes the unfaving of a given status, returning the updated status if the fave goes through.
	StatusUnfave(authed *oauth.Auth, targetStatusID string) (*apimodel.Status, error)

	// HomeTimelineGet returns statuses from the home timeline, with the given filters/parameters.
	HomeTimelineGet(authed *oauth.Auth, maxID string, sinceID string, minID string, limit int, local bool) ([]apimodel.Status, ErrorWithCode)

	/*
		FEDERATION API-FACING PROCESSING FUNCTIONS
		These functions are intended to be called when the federating client needs an immediate (ie., synchronous) reply
		to an HTTP request. As such, they will only do the bare-minimum of work necessary to give a properly
		formed reply. For more intensive (and time-consuming) calls, where you don't require an immediate
		response, pass work to the processor using a channel instead.
	*/

	// GetFediUser handles the getting of a fedi/activitypub representation of a user/account, performing appropriate authentication
	// before returning a JSON serializable interface to the caller.
	GetFediUser(requestedUsername string, request *http.Request) (interface{}, ErrorWithCode)

	// GetFediFollowers handles the getting of a fedi/activitypub representation of a user/account's followers, performing appropriate
	// authentication before returning a JSON serializable interface to the caller.
	GetFediFollowers(requestedUsername string, request *http.Request) (interface{}, ErrorWithCode)

	// GetFediFollowing handles the getting of a fedi/activitypub representation of a user/account's following, performing appropriate
	// authentication before returning a JSON serializable interface to the caller.
	GetFediFollowing(requestedUsername string, request *http.Request) (interface{}, ErrorWithCode)

	// GetFediStatus handles the getting of a fedi/activitypub representation of a particular status, performing appropriate
	// authentication before returning a JSON serializable interface to the caller.
	GetFediStatus(requestedUsername string, requestedStatusID string, request *http.Request) (interface{}, ErrorWithCode)

	// GetWebfingerAccount handles the GET for a webfinger resource. Most commonly, it will be used for returning account lookups.
	GetWebfingerAccount(requestedUsername string, request *http.Request) (*apimodel.WebfingerAccountResponse, ErrorWithCode)

	// InboxPost handles POST requests to a user's inbox for new activitypub messages.
	//
	// InboxPost returns true if the request was handled as an ActivityPub POST to an actor's inbox.
	// If false, the request was not an ActivityPub request and may still be handled by the caller in another way, such as serving a web page.
	//
	// If the error is nil, then the ResponseWriter's headers and response has already been written. If a non-nil error is returned, then no response has been written.
	//
	// If the Actor was constructed with the Federated Protocol enabled, side effects will occur.
	//
	// If the Federated Protocol is not enabled, writes the http.StatusMethodNotAllowed status code in the response. No side effects occur.
	InboxPost(ctx context.Context, w http.ResponseWriter, r *http.Request) (bool, error)
}

// processor just implements the Processor interface
type processor struct {
	// federator     pub.FederatingActor
	// toClientAPI   chan gtsmodel.ToClientAPI
	fromClientAPI chan gtsmodel.FromClientAPI
	// toFederator   chan gtsmodel.ToFederator
	fromFederator chan gtsmodel.FromFederator
	federator     federation.Federator
	stop          chan interface{}
	log           *logrus.Logger
	config        *config.Config
	tc            typeutils.TypeConverter
	oauthServer   oauth.Server
	mediaHandler  media.Handler
	storage       storage.Storage
	db            db.DB
}

// NewProcessor returns a new Processor that uses the given federator and logger
func NewProcessor(config *config.Config, tc typeutils.TypeConverter, federator federation.Federator, oauthServer oauth.Server, mediaHandler media.Handler, storage storage.Storage, db db.DB, log *logrus.Logger) Processor {
	return &processor{
		// toClientAPI:   make(chan gtsmodel.ToClientAPI, 100),
		fromClientAPI: make(chan gtsmodel.FromClientAPI, 100),
		// toFederator:   make(chan gtsmodel.ToFederator, 100),
		fromFederator: make(chan gtsmodel.FromFederator, 100),
		federator:     federator,
		stop:          make(chan interface{}),
		log:           log,
		config:        config,
		tc:            tc,
		oauthServer:   oauthServer,
		mediaHandler:  mediaHandler,
		storage:       storage,
		db:            db,
	}
}

// func (p *processor) ToClientAPI() chan gtsmodel.ToClientAPI {
// 	return p.toClientAPI
// }

func (p *processor) FromClientAPI() chan gtsmodel.FromClientAPI {
	return p.fromClientAPI
}

// func (p *processor) ToFederator() chan gtsmodel.ToFederator {
// 	return p.toFederator
// }

func (p *processor) FromFederator() chan gtsmodel.FromFederator {
	return p.fromFederator
}

// Start starts the Processor, reading from its channels and passing messages back and forth.
func (p *processor) Start() error {
	go func() {
	DistLoop:
		for {
			select {
			case clientMsg := <-p.fromClientAPI:
				p.log.Infof("received message FROM client API: %+v", clientMsg)
				if err := p.processFromClientAPI(clientMsg); err != nil {
					p.log.Error(err)
				}
			case federatorMsg := <-p.fromFederator:
				p.log.Infof("received message FROM federator: %+v", federatorMsg)
				if err := p.processFromFederator(federatorMsg); err != nil {
					p.log.Error(err)
				}
			case <-p.stop:
				break DistLoop
			}
		}
	}()
	return nil
}

// Stop stops the processor cleanly, finishing handling any remaining messages before closing down.
// TODO: empty message buffer properly before stopping otherwise we'll lose federating messages.
func (p *processor) Stop() error {
	close(p.stop)
	return nil
}
