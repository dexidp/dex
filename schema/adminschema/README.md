
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

### ClientCreateRequest

'client' field is a client registration request as defined by the OpenID Connect dynamic registration spec, and holds fields such as redirect URLs, prefered algorithms, etc. For brevity field names and types of that object have been omitted.

```
{
    client: {
    },
    isAdmin: boolean
}
```

### ClientRegistrationResponse

This object is a client registration respones as defined by the OpenID Connect dynamic registration spec. For brevity field names and types have been omitted.

```
{
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
| 200 |  | [ClientRegistrationResponse](#clientregistrationresponse) |
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


