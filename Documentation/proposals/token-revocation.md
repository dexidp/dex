# Proposal: design for revoking refresh tokens.

Refresh tokens are issued to the client by the authorization server and are used
to request a new access token when the current access token becomes invalid or expires.
It is a common usecase for the end users to revoke client access to their identity.
This proposal defines the changes needed in Dex v2 to support refresh token revocation.

## Motivation

1. Currently refresh tokens are not associated with the user. Need a new "session object" for this.
2. Need an API to list refresh tokens based on the UserID.
3. We need a way for users to login to dex and revoke a client.
4. Limit the number refresh tokens for each user-client pair to 1.

## Details

Currently in Dex when an end user successfully logs in via a connector and has the OfflineAccess
scope set to true, a refresh token is created and stored in the backing datastore. There is no
association between the end user and the refresh token. Hence if we want to support the functionality
of users being able to revoke refresh tokens, the first step is to have a structure in place that allows
us retrieve a list of refresh tokens depending on the authenticated user.

```go
// Reference object for RefreshToken containing only metadata.
type RefreshTokenRef struct {
	// ID of the RefreshToken
	ID string
	CreatedAt time.Time
	LastUsed  time.Time
}

// Session objects pertaining to users with refresh tokens.
//
// Will have to handle garbage collection i.e. if no refresh token exists for a user,
// this object must be cleaned up.
type OfflineSession struct {
        // UserID of an end user who has logged in to the server.
        UserID        string
        // The ID of the connector used to login the user.
        ConnID     string
        // List of pointers to RefreshTokens issued for SessionID
        Refresh         []*RefreshTokenRef
}

// Retrieve OfflineSession obj for given userId and connID
func getOfflineSession (userId string, connID string)

```

### Changes in Dex CodeFlows

1. Client requests a refresh token:
   Try to retrieve the `OfflineSession` object for the User with the given `UserID + ConnID`.
   This leads to two possibilities:   
	* Object exists: This means a Refresh token already exists for the user.
          Update the existing `OffilineSession` object with the newly received token as follows:
		* CreateRefresh() will create a new `RefreshToken` obj in the storage.
		* Update the `Refresh` list with the new `RefreshToken` pointer.
		* Delete the old refresh token in storage.

	* No object found: This implies that this will be the first refresh token for the user.
 		* CreateRefresh() will create a new `RefreshToken` obj in the storage.
		* Create an OfflineSession for the user and add the new `RefreshToken` pointer to
		  the `Refresh` list.
                
2. Refresh token rotation:
   There will be no change to this codeflow. When the client refreshes a refresh token, the `TokenID`
   still remains intact and only the `RefreshToken` obj gets updated with a new nonce. We do not need
   any additional checks in the OfflineSession objects as the `RefreshToken` pointers still remain intact.

3. User revokes a refresh token (New functionality):
   A user that has been authenticated externally will have the ability to revoke their refresh tokens.
   Please note that Dex's API does not perform the authentication, this will have to be done by an
   external app.
   Steps involved:
	* Get `OfflineSession` obj with given UserID + ConnID. 
	* If a refresh token exists in `Refresh`, delete the `RefreshToken` (handle this in storage)
	  and its pointer value in `Refresh`. Clean up the OfflineSession object.
	* If there is no refresh token found, handle error case.

NOTE: To avoid race conditions between “requesting a refresh token” and “revoking a refresh token”, use
locking mechanism when updating an `OfflineSession` object.
