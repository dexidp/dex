# Additional features

## Roles layer

### Preface
In most cases the authentication is used against an existing federation, that already contains users. Usually
an application has multiple views and access levels the user access might be managed by the user
federation. When you would like to use multiple user federations this management of access uis getting more
complex. This level of access is usually done by the user federation attributes or the group membership.
But to bind the access of a user direct to the strict group name is quite inflexible. When you try this for
multiple user federations for the same application it is in nearly impossible to manage this with the same groups.

The original Dex by default only supports to get all groups of a federation as claim when using the scope ***Groups***.

Here this feature of the  ***roles layer*** comes in.

### Description

Lets start with an example. You do need to provide for an application an authentication by three user federations.
One internal Active Directory used by the LDAP connector, one by Azure cloud and the last one by Github.
Each federation has its own group structure. How you ever will be able to manage the access for the contents of an
application?

By this roles layer this is quite simple. For each federation connector you can add by its configuration
a list of groups, where each group is correlated to a liszt of roles.

:hammer:
example:
* Group1:
    * frontend
* Group2:
    * admin
* Group3:
    * frontend
    * admin

<br>
:info:
* if the user is member of the group "Group1" the role "frontend" are added to the identity.
* if the user is member of the group "Group2" the role "admin" are added to the identity.
* if the user is member of the group "Group1" the roles "frontend" and "admin" are added to the identity.

Because you may add this role configurations for each connector you can apply the individual groups of each federation
to your application roles, that will finally be translted to the distinct roles.

:warning:
Do not forget to use the scope **Roles** to get the roles claim.


### Configuration

### Scope

if you need to get the translated roles, that are translated from the user group memberships you need to
set the scope **_Roles_**.

