
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


