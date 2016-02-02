
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


