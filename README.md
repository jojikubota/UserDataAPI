## Assignment

You will be building a simple RESTful service to manage personal perferences that can be shared with service providers like travel agencies so that they can provide personalization in their services.


### APIs
1. Create new user profile.

__URL: POST /profile__

***Request***
```json
{
"email": "foo@gmail.com",
"zip": "94112",
"country": "U.S.A",
"profession": "student",
"favorite_color": "blue",
"is_smoking": "yes|no",
"favorite_sport": "hiking",
"food": {
"type": "vegetrian|meat_eater|eat_everything",
"drink_alcohol": "yes|no"
},
"music": {
"spotify_user_id": "wizzler"
},
"movie": {
"tv_shows": ["x", "y", "z"],
"movies": ["x", "y", "z"]
},
"travel": {
"flight": {
"seat": "aisle|window"            
}
}
}
``` 
***Response***
```sh
HTTP/1.1 201 Created
Date: Mon, 29 Feb 2016 19:55:15 GMT
....
``` 
---
2. Get user profile.

__URL: GET /profile/{email}__

***Response***
```sh
HTTP/1.1 200 OK
Date: Mon, 29 Feb 2016 19:55:15 GMT
....
``` 

```json
{
"email": "foo@gmail.com",
"zip": "94112",
"country": "U.S.A",
"profession": "student",
"favorite_color": "blue",
"is_smoking": "yes",
"favorite_sport": "hiking",
"food": {
"type": "meat_eater",
"drink_alcohol": "no"
},
"music": {
"spotify_user_id": "wizzler"
},
"movie": {
"tv_shows": ["x", "y", "z"],
"movies": ["x", "y", "z"]
},
"travel": {
"flight": {
"seat": "aisle|window"            
}
}
}
``` 
---
3. Update existing user profile.

__URL: PUT /profile/{email}__

***Request***
```json
{
"travel": {
"flight": {
"seat": "window"            
}
}
}
``` 

*** Response ***
```sh
HTTP/1.1 204 No Content
Date: Mon, 29 Feb 2016 19:55:15 GMT
....
``` 
---
4. Delete user profile.

__URL: DELETE /profile/{email}__

***Response***
```sh
HTTP/1.1 204 No Content
Date: Mon, 29 Feb 2016 19:55:15 GMT
....
``` 
---


#### Data Persistence
* All the CRUD APIs will be now storing and retrieving data from [EJDB](http://ejdb.org/doc/snippets.html#go).

#### Configuration

* Use the following [TOML](https://github.com/toml-lang/toml) (app*.toml) file to externalize all configurations. 
* Implement TOML configuration so that you can pass in TOML file from the command line.

```sh
go run app.go app1.toml
```

* app1.toml
```toml
[database]
file_name = "app1.db"

# REST API Port
port_num = 4001

[replication]
rpc_server_port_num = 3001
replica = [ "http://0.0.0.0:3002" ]
```

* app2.toml
```toml
[database]
file_name = "app2.db"

# REST API Port
port_num = 4002

[replication]
rpc_server_port_num = 3002
replica = [ "http://0.0.0.0:3001" ]
```

> [How to toml?](https://github.com/naoina/toml/tree/master/_example) 

#### Replication
 * Implement data replication for all EJDB write (Insert and Update) queries to all replica via [RPC over TCP](https://gist.github.com/jordanorelli/2629049).
 * You can do query replication either in JSON or BSON.
 * This replication must be bi-directional so that both RPC client and server's implementation is required on a server. You will be running the same code for all instances with different configuration toml files.
 * Since the main thread is listening on the REST API requests, you have to use go routine to launch RPC (server) listener. 
 * Use app1.toml for server 1 and app2.toml for another one.
 * Your code must work for any number of replica.
 
#### EJDB Alternative

If you cannot install EJDB for whatever reason, you are allowed to use SQLite3 as database. You don't need to install anything for SQLite other than just this go-get:

```sh
go get github.com/mattn/go-sqlite3
```

You can find more examples at [Sqlite's github](https://github.com/mattn/go-sqlite3) in addition to this [sample code](https://github.com/sithu/cmpe273-sp16/blob/master/assignment2/app_sqlite3.go).
 
