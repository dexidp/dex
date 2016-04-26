
# Dex API

The Dex REST API

__Version:__ v1

## Models


### Client



```
{
    id: string,
    redirectURIs: [
        string
    ]
}
```

### ClientPage



```
{
    clients: [
        Client
    ],
    nextPageToken: string
}
```

### ClientWithSecret



```
{
    id: string,
    redirectURIs: [
        string
    ],
    secret: string
}
```

### Error



```
{
    error: string,
    error_description: string
}
```

### RefreshClient

A client with associated public metadata.

```
{
    clientID: string,
    clientName: string,
    clientURI: string,
    logoURI: string
}
```

### RefreshClientList



```
{
    clients: [
        RefreshClient
    ]
}
```

### ResendEmailInvitationRequest



```
{
    redirectURL: string
}
```

### ResendEmailInvitationResponse



```
{
    emailSent: boolean,
    resetPasswordLink: string
}
```

### User



```
{
    admin: boolean,
    createdAt: string,
    disabled: boolean,
    displayName: string,
    email: string,
    emailVerified: boolean,
    id: string
}
```

### UserCreateRequest



```
{
    redirectURL: string,
    user: User
}
```

### UserCreateResponse



```
{
    emailSent: boolean,
    resetPasswordLink: string,
    user: User
}
```

### UserDisableRequest



```
{
    disable: boolean // If true, disable this user, if false, enable them. No error is signaled if the user state doesn't change.
}
```

### UserDisableResponse



```
{
    ok: boolean
}
```

### UserResponse



```
{
    user: User
}
```

### UsersResponse



```
{
    nextPageToken: string,
    users: [
        User
    ]
}
```


## Paths


### GET /account/{userid}/refresh

> __Summary__

> List RefreshClient

> __Description__

> List all clients that hold refresh tokens for the specified user.


> __Parameters__

> |Name|Located in|Description|Required|Type|
|:-----|:-----|:-----|:-----|:-----|
| userid | path |  | Yes | string | 


> __Responses__

> |Code|Description|Type|
|:-----|:-----|:-----|
| 200 |  | [RefreshClientList](#refreshclientlist) |
| default | Unexpected error |  |


### DELETE /account/{userid}/refresh/{clientid}

> __Summary__

> Revoke RefreshClient

> __Description__

> Revoke all refresh tokens issues to the client for the specified user.


> __Parameters__

> |Name|Located in|Description|Required|Type|
|:-----|:-----|:-----|:-----|:-----|
| clientid | path |  | Yes | string | 
| userid | path |  | Yes | string | 


> __Responses__

> |Code|Description|Type|
|:-----|:-----|:-----|
| default | Unexpected error |  |


### GET /clients

> __Summary__

> List Clients

> __Description__

> Retrieve a page of Client objects.


> __Parameters__

> |Name|Located in|Description|Required|Type|
|:-----|:-----|:-----|:-----|:-----|
| nextPageToken | query |  | No | string | 


> __Responses__

> |Code|Description|Type|
|:-----|:-----|:-----|
| 200 |  | [ClientPage](#clientpage) |
| default | Unexpected error |  |


### POST /clients

> __Summary__

> Create Clients

> __Description__

> Register a new Client.


> __Parameters__

> |Name|Located in|Description|Required|Type|
|:-----|:-----|:-----|:-----|:-----|
|  | body |  | Yes | [Client](#client) | 


> __Responses__

> |Code|Description|Type|
|:-----|:-----|:-----|
| 200 |  | [ClientWithSecret](#clientwithsecret) |
| default | Unexpected error |  |


### GET /users

> __Summary__

> List Users

> __Description__

> Retrieve a page of User objects.


> __Parameters__

> |Name|Located in|Description|Required|Type|
|:-----|:-----|:-----|:-----|:-----|
| nextPageToken | query |  | No | string | 
| maxResults | query |  | No | integer | 


> __Responses__

> |Code|Description|Type|
|:-----|:-----|:-----|
| 200 |  | [UsersResponse](#usersresponse) |
| default | Unexpected error |  |


### POST /users

> __Summary__

> Create Users

> __Description__

> Create a new User.


> __Parameters__

> |Name|Located in|Description|Required|Type|
|:-----|:-----|:-----|:-----|:-----|
|  | body |  | Yes | [UserCreateRequest](#usercreaterequest) | 


> __Responses__

> |Code|Description|Type|
|:-----|:-----|:-----|
| 200 |  | [UserCreateResponse](#usercreateresponse) |
| default | Unexpected error |  |


### GET /users/{id}

> __Summary__

> Get Users

> __Description__

> Get a single User object by id.


> __Parameters__

> |Name|Located in|Description|Required|Type|
|:-----|:-----|:-----|:-----|:-----|
| id | path |  | Yes | string | 


> __Responses__

> |Code|Description|Type|
|:-----|:-----|:-----|
| 200 |  | [UserResponse](#userresponse) |
| default | Unexpected error |  |


### POST /users/{id}/disable

> __Summary__

> Disable Users

> __Description__

> Enable or disable a user.


> __Parameters__

> |Name|Located in|Description|Required|Type|
|:-----|:-----|:-----|:-----|:-----|
| id | path |  | Yes | string | 
|  | body |  | Yes | [UserDisableRequest](#userdisablerequest) | 


> __Responses__

> |Code|Description|Type|
|:-----|:-----|:-----|
| 200 |  | [UserDisableResponse](#userdisableresponse) |
| default | Unexpected error |  |


### POST /users/{id}/resend-invitation

> __Summary__

> ResendEmailInvitation Users

> __Description__

> Resend invitation email to an existing user with unverified email.


> __Parameters__

> |Name|Located in|Description|Required|Type|
|:-----|:-----|:-----|:-----|:-----|
| id | path |  | Yes | string | 
|  | body |  | Yes | [ResendEmailInvitationRequest](#resendemailinvitationrequest) | 


> __Responses__

> |Code|Description|Type|
|:-----|:-----|:-----|
| 200 |  | [ResendEmailInvitationResponse](#resendemailinvitationresponse) |
| default | Unexpected error |  |


