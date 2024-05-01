/*
	GoToSocial
	Copyright (C) GoToSocial Authors admin@gotosocial.org
	SPDX-License-Identifier: AGPL-3.0-or-later

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

import React, { ReactNode } from "react";
import { useSearchAccountsQuery } from "../../../../lib/query/admin";
import { PageableList } from "../../../../components/pageable-list";
import { useLocation } from "wouter";
import Username from "../../../../components/username";
import { AdminAccount } from "../../../../lib/types/account";

export default function AccountsPending() {
	const [ location, _setLocation ] = useLocation();
	const searchRes = useSearchAccountsQuery({status: "pending"});

	// Function to map an item to a list entry.
	function itemToEntry(account: AdminAccount): ReactNode {
		const acc = account.account;
		return (
			<Username
				key={acc.acct}
				account={account}
				linkTo={`/${account.id}`}
				backLocation={location}
				classNames={["entry"]}
			/>
		);
	}

	return (
		<div className="accounts-view">
			<h1>Pending Accounts</h1>
			<PageableList
				isLoading={searchRes.isLoading}
				isFetching={searchRes.isFetching}
				isSuccess={searchRes.isSuccess}
				items={searchRes.data?.accounts}
				itemToEntry={itemToEntry}
				isError={searchRes.isError}
				error={searchRes.error}
				emptyMessage="No pending account sign-ups."
			/>
		</div>
	);
}
