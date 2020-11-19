# first-auth CLI

The main goal of this project is to interact with gRPC api function in Dex (see __ReadMe.md__ into the gRPCapi folder) and create a CLI to be able to manage first authentification database fo Dex.

This Project use the package Cobra to generate appropriate command with flags. See below all command allowed

The name of the command line is **first-auth**

## Credentials files

Firstly you need to generate credentials files with dex and give the path of **ca.crt**, **client.crt** and **client.ca** files, like below:

```bash
./bin/first-auth <command> 
    --ca-crt=path/to/your/ca.crt
    --client-crt=path/to/your/client.crt
    --client-key=path/to/your/client.key
```

Then you can choose between each commands

## Commands with flags

| Commmands                         |      Flags                |  Explanation      |
|-----------------------------------|:-------------------------:|:------------------|
| ./bin/first-auth genUserIdp       | --userID<br>--pseudo<br>--email<br>--aclTokens    | generate an idp user and an user linked with acl tokens given     |
| ./bin/first-auth upUserIdp        | --userID<br>--pseudo<br>--email<br>--aclTokens    | update the idp user and the internal user                         |
| ./bin/first-auth delUserIdp       | --userID                                          | delete the idp user                                               |
|-|-|-|
| ./bin/first-auth delUser          | --internalID              | delete the user   |
|-|-|-|
| ./bin/first-auth genAclToken      | --desc<br>--maxUSer<br>--clientTokens                 | generate an acl token linked with client tokens given     |
| ./bin/first-auth upAclToken       | --tokenID<br>--desc<br>--maxUSer<br>--clientTokens    | update the acl token                                      |
| ./bin/first-auth delAclToken      | --tokenID                                             | delete the acl token                                      |
|-|-|-|
| ./bin/first-auth genClientToken      | --clientID<br>--days                 | generate a client token an give an expiration with the number of days given (can be negative if need to sorter the expiration)     |
| ./bin/first-auth upClientToken       | --tokenID<br>--clientID<br>--days    | update the client token                                      |
| ./bin/first-auth delClientToken      | --tokenID                            | delete the acl token                                         |


### Example of this command

Not available