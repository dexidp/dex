
# Dex Admin API

The Dex Admin API.

__Version:__ v1

## Models


### Admin



```
{
    email: string,
    id: string,
    password: string
}
```

### Client



```
{
    clientName: string // OPTIONAL. Name of the Client to be presented to the End-User. If desired, representation of this Claim in different languages and scripts is represented as described in Section 2.1 ( Metadata Languages and Scripts ) .,
    clientURI: string // OPTIONAL. URL of the home page of the Client. The value of this field MUST point to a valid Web page. If present, the server SHOULD display this URL to the End-User in a followable fashion. If desired, representation of this Claim in different languages and scripts is represented as described in Section 2.1 ( Metadata Languages and Scripts ) .,
    id: string // The client ID. Ignored in client create requests.,
    isAdmin: boolean,
    logoURI: string // OPTIONAL. URL that references a logo for the Client application. If present, the server SHOULD display this image to the End-User during approval. The value of this field MUST point to a valid image file. If desired, representation of this Claim in different languages and scripts is represented as described in Section 2.1 ( Metadata Languages and Scripts ) .,
    redirectURIs: [
        string
    ],
    secret: string // The client secret. Ignored in client create requests.
}
```

### ClientCreateRequest

A request to register a client with dex.

```
{
    client: Client
}
```

### ClientCreateResponse

Upon successful registration, an ID and secret is assigned to the client.

```
{
    client: Client
}
```

### State



```
{
    AdminUserCreated: boolean
}
```


## Paths


### POST /admin

> __Summary__

> Create Admin

> __Description__

> Create a new admin user.


> __Parameters__

> |Name|Located in|Description|Required|Type|
|:-----|:-----|:-----|:-----|:-----|
|  | body |  | Yes | [Admin](#admin) | 


> __Responses__

> |Code|Description|Type|
|:-----|:-----|:-----|
| 200 |  | [Admin](#admin) |
| default | Unexpected error |  |


### GET /admin/{id}

> __Summary__

> Get Admin

> __Description__

> Retrieve information about an admin user.


> __Parameters__

> |Name|Located in|Description|Required|Type|
|:-----|:-----|:-----|:-----|:-----|
| id | path |  | Yes | string | 


> __Responses__

> |Code|Description|Type|
|:-----|:-----|:-----|
| 200 |  | [Admin](#admin) |
| default | Unexpected error |  |


### POST /client

> __Summary__

> Create Client

> __Description__

> Register an OpenID Connect client.


> __Parameters__

> |Name|Located in|Description|Required|Type|
|:-----|:-----|:-----|:-----|:-----|
|  | body |  | Yes | [ClientCreateRequest](#clientcreaterequest) | 


> __Responses__

> |Code|Description|Type|
|:-----|:-----|:-----|
| 200 |  | [ClientCreateResponse](#clientcreateresponse) |
| default | Unexpected error |  |


### GET /state

> __Summary__

> Get State

> __Description__

> Get the state of the Dex DB


> __Responses__

> |Code|Description|Type|
|:-----|:-----|:-----|
| 200 |  | [State](#state) |
| default | Unexpected error |  |


