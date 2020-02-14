/*
	grp contains methods to parse the various group-formats for
	ActiveDirectory and UNIX LDAP.

	ActiveDirectory: 	TnPg_Srv_Appkaas-clustername-namespace-role_full
	UNIX-LDAP:			tnnt_kaas-clustername-namespace-role

	Tn, tnnt are the tenant-prefixes

	For group policies all that matters are the elements of the stripped
    "inner" group-name, in this case "clustername", "namespace", "role"
*/
package grp
