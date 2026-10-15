/** Everything a request runs on behalf of: the tenant, and who is asking. */
export interface Scope {
	/**
	 * The tenant whose data this is. People an account is shared with act in
	 * the owner's tenant, so this is not always the person's own.
	 */
	tenantId: string;
	accountId: string;
}
