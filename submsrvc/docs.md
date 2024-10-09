
INSERT statement is used to insert a single record or multiple records into a table.  
More about INSERT statement can be at:  
PostgreSQL - https://www.postgresql.org/docs/11/sql-insert.html  
MySQL - https://dev.mysql.com/doc/refman/8.0/en/insert.html  
MariaDB - https://mariadb.com/kb/en/library/update/  

Following clauses are supported:
- `INSERT(columns...)` - list of columns for insert
- `VALUES(values...)` - list of values 
- `MODEL(model)` - list of values for columns will be extracted from model object
- `MODELS([]model)`  - list of values for columns will be extracted from list of model objects
- `QUERY(select)` - select statement that supplies the rows to be inserted.
- `ON CONFLICT` - specifies an alternative action to raising a unique violation or exclusion constraint violation error (PostgreSQL only).
- `ON DUPLICATE KEY UPDATE` - enables existing rows to be updated if a row to be inserted would cause a 
duplicate value in a UNIQUE index or PRIMARY KEY(MySQL and MariaDB).
- `RETURNING(output_expression...)` - An expressions to be computed and returned by the INSERT statement after each row is inserted.
The expressions can use any column names of the table. Use _TableName_.AllColumns to return all columns. (PostgreSQL only)


_This list might be extended with feature Jet releases._ 

### Insert row by row

#### Using VALUES (not recommended, see bellow)
```golang
insertStmt := Link.INSERT(Link.ID, Link.URL, Link.Name, Link.Description).
    VALUES(100, "http://www.postgresqltutorial.com", "PostgreSQL Tutorial", DEFAULT).
    VALUES(101, "http://www.google.com", "Google", DEFAULT).
    VALUES(102, "http://www.yahoo.com", "Yahoo", nil)
```
Debug SQL of above insert statement:

```sql
INSERT INTO test_sample.link (id, url, name, description) VALUES
     (100, 'http://www.postgresqltutorial.com', 'PostgreSQL Tutorial', DEFAULT),
     (101, 'http://www.google.com', 'Google', DEFAULT),
     (102, 'http://www.yahoo.com', 'Yahoo', NULL)
```


#### Using MODEL, MODELS (recommended)
This notation is recommended, because model types will add type and pointer safety to insert query.

```golang
tutorial := model.Link{
    ID:   100,
    URL:  "http://www.postgresqltutorial.com",
    Name: "PostgreSQL Tutorial",
}

google := model.Link{
    ID:   101,
    URL:  "http://www.google.com",
    Name: "Google",
}

yahoo := model.Link{
    ID:   102,
    URL:  "http://www.yahoo.com",
    Name: "Yahoo",
}

insertStmt := Link.INSERT(Link.ID, Link.URL, Link.Name, Link.Description).
    MODEL(turorial).
    MODEL(google).
    MODEL(yahoo)
```
Or event shorter if model data is in the slice:
```golang
insertStmt := Link.INSERT(Link.ID, Link.URL, Link.Name, Link.Description).
    MODELS([]model.Link{turorial, google, yahoo})
```
`Link.ID, Link.URL, Link.Name, Link.Description` - is the same as `Link.AllColumns` 
so above statement can be simplified to:

```golang
insertStmt := Link.INSERT(Link.AllColumns).
    MODELS([]model.Link{turorial, google, yahoo})
```

`Link.ID` is a primary key autoincrement column so it can be omitted in INSERT statement.  
`Link.MutableColumns` - is shorthand notation for list of all columns minus primary key columns.

```golang
insertStmt := Link.INSERT(Link.MutableColumns).
    MODELS([]model.Link{turorial, google, yahoo})
```

`ColumnList` can be used to pass a custom list of columns to the INSERT query:
```golang
columnList := ColumnList{Link.Name, Link.Description}
insertStmt := Link.INSERT(columnList).
    MODEL(turorial)
```

Inserts using `VALUES`, `MODEL` and `MODELS` can appear as the part of the same insert statement.

```golang
insertStmt := Link.INSERT(Link.ID, Link.URL, Link.Name, Link.Description).
    VALUES(101, "http://www.google.com", "Google", DEFAULT, DEFAULT).
    MODEL(turorial).
    MODELS([]model.Link{yahoo})
``` 

### Insert using query
```golang
// duplicate first 10 entries
insertStmt := Link.
    INSERT(Link.URL, Link.Name).
    QUERY(
        SELECT(Link.URL, Link.Name).
            FROM(Link).
            WHERE(Link.ID.GT(Int(0)).AND(Link.ID.LT_EQ(10))),
    )
```
### Upsert

#### [PostgreSQL, SQLite] Insert with ON CONFLICT update

- ON CONFLICT DO NOTHING
```golang
Employee.INSERT(Employee.AllColumns).
MODEL(employee).
ON_CONFLICT(Employee.EmployeeID).DO_NOTHING()
```

- ON CONFLICT DO UPDATE
```golang
Link.INSERT(Link.ID, Link.URL, Link.Name, Link.Description).
VALUES(100, "http://www.postgresqltutorial.com", "PostgreSQL Tutorial", DEFAULT).
ON_CONFLICT(Link.ID).DO_UPDATE(
    SET(
        Link.ID.SET(Link.EXCLUDED.ID),
        Link.URL.SET(String("http://www.postgresqltutorial2.com")),
    ),
)
```

- ON CONFLICT DO UPDATE WHERE
```golang
Link.INSERT(Link.ID, Link.URL, Link.Name, Link.Description).
VALUES(100, "http://www.postgresqltutorial.com", "PostgreSQL Tutorial", DEFAULT).
ON_CONFLICT(Link.ID).
    WHERE(Link.ID.MUL(Int(2)).GT(Int(10))).
    DO_UPDATE(
        SET(
            Link.ID.SET(
                IntExp(SELECT(MAXi(Link.ID).ADD(Int(1))).
                    FROM(Link)),
            ),
            ColumnList{Link.Name, Link.Description}.SET(ROW(Link.EXCLUDED.Name, String("new description"))),
        ).WHERE(Link.Description.IS_NOT_NULL()),
    )
```

#### [MySQL] Insert with ON DUPLICATE KEY UPDATE
```golang
Link.INSERT().
	VALUES(randId, "http://www.postgresqltutorial.com", "PostgreSQL Tutorial", DEFAULT).
	ON_DUPLICATE_KEY_UPDATE(
    		Link.ID.SET(Link.ID.ADD(Int(11))),
    		Link.Name.SET(String("PostgreSQL Tutorial 2")),
	)
```
- New rows aliased
```golang
Link.INSERT().
	MODEL(model.Link{
		{
			ID:          randId,
			URL:         "https://www.postgresqltutorial.com",
			Name:        "PostgreSQL Tutorial",
			Description: nil,
		},
	}).AS_NEW().      // Note !!! 
	ON_DUPLICATE_KEY_UPDATE(
		Link.URL.SET(Link.NEW.URL),
		Link.Name.SET(Link.NEW.Name),
	)
```

## Execute statement

To execute insert statement and get sql.Result:

```golang
res, err := insertStmt.Exec(db)
```

To execute insert statement and return records inserted, insert statement has to have RETURNING clause:
```golang
insertStmt := Link.INSERT(Link.ID, Link.URL, Link.Name, Link.Description).
    VALUES(100, "http://www.postgresqltutorial.com", "PostgreSQL Tutorial", DEFAULT).
    VALUES(101, "http://www.google.com", "Google", DEFAULT).
    RETURNING(Link.ID, Link.URL, Link.Name, Link.Description)  // or RETURNING(Link.AllColumns)
    
dest := []model.Link{}

err := insertStmt.Query(db, &dest)
```

Use `ExecContext` and `QueryContext` to provide context object to execution.

##### SQL table used for the example:
```sql
CREATE TABLE IF NOT EXISTS link (
    id serial PRIMARY KEY,
    url VARCHAR (255) NOT NULL,
    name VARCHAR (255) NOT NULL,
    description VARCHAR (255)
);
```

Model file contains simple Go struct type used to store result of SQL queries. Model types can be used alone or combined to form complex object composition to store database query result. They are auto-generated from database tables, views and enums. 

### Table and view model files

Following rules are applied to generate model types from database tables and views:

- for every table there is one Go file generated. File name is in snake case of the table name. 
- every model file contains one struct type. Type name is a camel case of table name. Package name
is always `model`.
- for every column of table there is a field in model struct type. Field name is camel case of column name. 
See below table for type mapping.
- fields are pointer types, if they relate to column that can be NULL. 
- fields corresponds to primary key columns are tagged with `sql:"primary_key"`.
This tag is used during query execution to group row results into desired arbitrary structure. 
See more at [Query Result Mapping (QRM)](https://github.com/go-jet/jet/wiki/Query-Result-Mapping-(QRM))  

##### Mappings of Postgres database types to Go types

| Database type(postgres)                         | Go type                                            |
| ----------------------------------------------- | -------------------------------------------------- |
| boolean                                         |  bool                                              |
| smallint                                        |  int16                                             |
| integer                                         |  int32                                             |
| bigint                                          |  int64                                             |
| real                                            |  float32                                           |
| numeric, decimal, double precision              |  float64                                           |
| date, timestamp, time(with or without timezone) |  time.Time                                         |
| bytea                                           |  []byte                                            |
| uuid                                            |  uuid.UUID                                         |
| enum                                            |  enum name                                         |
| text, character, character varying,             |                                                    |
| and all remaining types                         |  string                                            |

##### Mappings of MySQL and MariaDB database types to Go types

| Database type(postgres)                         | Go type                                            |
| ----------------------------------------------- | -------------------------------------------------- |
| boolean or BIT(1)                               |  bool                                              |
| tinylint [unsigned]                             |  [u]int8                                           |
| smallint [unsigned]                             |  [u]int16                                          |
| mediumint [unsigned]                            |  [u]int32                                          |
| integer [unsigned]                              |  [u]int32                                          |
| bigint  [unsigned]                              |  [u]int64                                          |
| real                                            |  float32                                           |
| numeric, decimal, double precision              |  float64                                           |
| date, time, datetime, timestamp                 |  time.Time                                         |
| binary, varbinary, tinyblob, blob,              |                                                    |
| mediumblob, longblob                            |  []byte                                            |
| enum                                            |  table name + enum name                            |
| text, character, character varying,             |                                                    |
| and all remaining types                         |  string                                            |


#### Example:

PostgreSQL table `address`:
```sql
CREATE TABLE dvds.address
(
    address_id serial NOT NULL DEFAULT,
    address character varying(50) NOT NULL,
    address2 character varying(50),
    district character varying(20) NOT NULL,
    city_id smallint NOT NULL,
    postal_code character varying(10),
    phone character varying(20) NOT NULL,
    last_update timestamp without time zone NOT NULL DEFAULT now(),
    CONSTRAINT address_pkey PRIMARY KEY (address_id)
)
```

Autogenerated model file `address.go`:

```go
package model

import (
    "time"
)

type Address struct {
    AddressID  int32 `sql:"primary_key"`
    Address    string
    Address2   *string
    District   string
    CityID     int16
    PostalCode *string
    Phone      string
    LastUpdate time.Time
}
```

### Enum model files

Following rules are applied to generate model files from database enums:

- for every enum there is one Go file generated. 
    - PostgreSQL: File name is a snake case of `enum name`.
    - MySQL or MariaDB: File name is snake case of `table name` + `enum name`. 
- every file contains one renamed string type. Type name is a camel case of enum name.
Package name is always `model`.
Enum type has two helper methods to: 
    - initialize correctly from database query result
    - easily convert enum to string.
- for every enum value there is one constant defined. 
Name of the constant is in format `{CamelCase(enum_name)}_{CamelCase(enum_value_name)}`.

#### Example

PostgreSQL:
```sql
CREATE TYPE dvds.mpaa_rating AS ENUM
    ('G', 'PG', 'PG-13', 'R', 'NC-17');
```

Autogenerated model file `mpaa_rating.go`

```go
package model

import "errors"

type MpaaRating string

const (
	MpaaRating_G    MpaaRating = "G"
	MpaaRating_Pg   MpaaRating = "PG"
	MpaaRating_Pg13 MpaaRating = "PG-13"
	MpaaRating_R    MpaaRating = "R"
	MpaaRating_Nc17 MpaaRating = "NC-17"
)

func (e *MpaaRating) Scan(value interface{}) error {
	if v, ok := value.(string); !ok {
		return errors.New("jet: Invalid data for MpaaRating enum")
	} else {
		switch string(v) {
		case "G":
			*e = MpaaRating_G
		case "PG":
			*e = MpaaRating_Pg
		case "PG-13":
			*e = MpaaRating_Pg13
		case "R":
			*e = MpaaRating_R
		case "NC-17":
			*e = MpaaRating_Nc17
		default:
			return errors.New("Inavlid data " + string(v) + "for MpaaRating enum")
		}

		return nil
	}
}

func (e MpaaRating) String() string {
	return string(e)
}
```

MySQL or MariaDB:
```sql
CREATE TABLE film (
  rating ENUM('G','PG','PG-13','R','NC-17') DEFAULT 'G'
)
```

```go
package model

import "errors"

type FilmRating string

const (
	FilmRating_G    FilmRating = "G"
	FilmRating_Pg   FilmRating = "PG"
	FilmRating_Pg13 FilmRating = "PG-13"
	FilmRating_R    FilmRating = "R"
	FilmRating_Nc17 FilmRating = "NC-17"
)

func (e *FilmRating) Scan(value interface{}) error {
	if v, ok := value.(string); !ok {
		return errors.New("jet: Invalid data for FilmRating enum")
	} else {
		switch string(v) {
		case "G":
			*e = FilmRating_G
		case "PG":
			*e = FilmRating_Pg
		case "PG-13":
			*e = FilmRating_Pg13
		case "R":
			*e = FilmRating_R
		case "NC-17":
			*e = FilmRating_Nc17
		default:
			return errors.New("jet: Inavlid data " + string(v) + "for FilmRating enum")
		}

		return nil
	}
}

func (e FilmRating) String() string {
	return string(e)
}
```


## Contents
- [How scan works?](https://github.com/go-jet/jet/wiki/Query-Result-Mapping-(QRM)#how-scan-works)
- [Custom model types](https://github.com/go-jet/jet/wiki/Query-Result-Mapping-(QRM)#custom-model-types)
    - [Anonymous custom types](https://github.com/go-jet/jet/wiki/Query-Result-Mapping-(QRM)#anonymous-custom-types)
    - [Tagging model type fields](https://github.com/go-jet/jet/wiki/Query-Result-Mapping-(QRM)#tagging-model-type-fields)
- [Combining autogenerated and custom model types](https://github.com/go-jet/jet/wiki/Query-Result-Mapping-(QRM)#combining-autogenerated-and-custom-model-types)
- [Specifying primary keys](https://github.com/go-jet/jet/wiki/Query-Result-Mapping-(QRM)#specifying-primary-keys)      
   

### How scan works?

`Query` and `QueryContext` statement methods perform scan and grouping of each row result to arbitrary destination structure.

- `Query(db qrm.DB, destination interface{}) error` - executes statements over database connection db(or transaction) and stores row results in `destination`.
- `QueryContext(db qrm.DB, context context.Context, destination interface{}) error` - executes statement with a context over database connection db(or transaction) and stores row result in `destination`.

Destination can be either a pointer to struct or a pointer to slice of structs.

The easiest way to understand how scan works is by an example.
Lets say we want to retrieve list of cities, with list of customers for each city, and address for each customer.
For simplicity we will narrow the choice to 'London' and 'York'.

```go
stmt := 
    SELECT(
        City.CityID, 
        City.City,
        Address.AddressID, 
        Address.Address,
        Customer.CustomerID, 
        Customer.LastName,
    ).FROM(
        City.
            INNER_JOIN(Address, Address.CityID.EQ(City.CityID)).
            INNER_JOIN(Customer, Customer.AddressID.EQ(Address.AddressID)).
    ).
    WHERE(City.City.EQ(String("London")).OR(City.City.EQ(String("York")))).
    ORDER_BY(City.CityID, Address.AddressID, Customer.CustomerID)
```

Debug sql of above statement:
```sql
SELECT city.city_id AS "city.city_id",
     city.city AS "city.city",
     address.address_id AS "address.address_id",
     address.address AS "address.address",
     customer.customer_id AS "customer.customer_id",
     customer.last_name AS "customer.last_name"
FROM dvds.city
     INNER JOIN dvds.address ON (address.city_id = city.city_id)
     INNER JOIN dvds.customer ON (customer.address_id = address.address_id)
WHERE (city.city = 'London') OR (city.city = 'York')
ORDER BY city.city_id, address.address_id, customer.customer_id;
```

**Note that every column is aliased by default. Format is "`table_name`.`column_name`".**

Above statement will produce following result set:

|_row_| city.city_id |   city.city      | address.address_id  |   address.address     | customer.customer_id | customer.last_name |
|---  | ------------ | -------------    | ------------------- | --------------------  | -------------------- | ------------------ |
|  _1_|          312 |	  "London"      |	             256  |	"1497 Yuzhou Drive"	  |                  252 |           "Hoffman"|
|  _2_|          312 |	  "London"      |	             517  |	"548 Uruapan Street"  |                  512 |           "Vines"  | 
|  _3_|          589 |	  "York"        |	             502  |	"1515 Korla Way"	  |                  497 |           "Sledge" |

Lets execute statement and scan result set to destination `dest`:
 ```go
var dest []struct {
    model.City

    Customers []struct{
        model.Customer

        Address model.Address
    }
}

err := stmt.Query(db, &dest)
 ```

Note that camel case of result set column names(aliases) is the same as `model type name`.`field name`. 
For instance `city.city_id` -> `City.CityID`. This is being used to find appropriate column for each destination model field.
It is not an error if there is not a column for each destination model field. Table and column names does not have
to be in snake case.
 
`Query` uses reflection to introspect destination type structure, and result set column names(aliases), to find appropriate destination field for result set column.
Every new destination struct object is cached by his and all the parents primary key. So grouping to work correctly at least table primary keys has to appear in query result set. If there is no primary key in a result set
row number is used as grouping condition(which is always unique).    
For instance, after row 1 is processed, two objects are stored to cache:
```
Key:                                        Object:
(City(312))                              -> (*struct { model.City; Customers []struct { model.Customer; Address model.Address } })
(City(312)),(Customer(252),Address(256)) -> (*struct { model.Customer; Address model.Address })
```
After row 2 processing only one new object is stored to cache, because city with city_id 312 is already in cache.
```
Key:                                        Object:
(City(312))                              -> pulled from cache
(City(312)),(Customer(512),Address(517)) -> (*struct { model.Customer; Address model.Address })
```

Lets print `dest` as a json, to visualize `Query` result:
 
 ```js
 [
 	{
 		"CityID": 312,
 		"City": "London",
 		"CountryID": 0,
 		"LastUpdate": "0001-01-01T00:00:00Z",
 		"Customers": [
 			{
 				"CustomerID": 252,
 				"StoreID": 0,
 				"FirstName": "",
 				"LastName": "Hoffman",
 				"Email": null,
 				"AddressID": 0,
 				"Activebool": false,
 				"CreateDate": "0001-01-01T00:00:00Z",
 				"LastUpdate": null,
 				"Active": null,
 				"Address": {
 					"AddressID": 256,
 					"Address": "1497 Yuzhou Drive",
 					"Address2": null,
 					"District": "",
 					"CityID": 0,
 					"PostalCode": null,
 					"Phone": "",
 					"LastUpdate": "0001-01-01T00:00:00Z"
 				}
 			},
 			{
 				"CustomerID": 512,
 				"StoreID": 0,
 				"FirstName": "",
 				"LastName": "Vines",
 				"Email": null,
 				"AddressID": 0,
 				"Activebool": false,
 				"CreateDate": "0001-01-01T00:00:00Z",
 				"LastUpdate": null,
 				"Active": null,
 				"Address": {
 					"AddressID": 517,
 					"Address": "548 Uruapan Street",
 					"Address2": null,
 					"District": "",
 					"CityID": 0,
 					"PostalCode": null,
 					"Phone": "",
 					"LastUpdate": "0001-01-01T00:00:00Z"
 				}
 			}
 		]
 	},
 	{
 		"CityID": 589,
 		"City": "York",
 		"CountryID": 0,
 		"LastUpdate": "0001-01-01T00:00:00Z",
 		"Customers": [
 			{
 				"CustomerID": 497,
 				"StoreID": 0,
 				"FirstName": "",
 				"LastName": "Sledge",
 				"Email": null,
 				"AddressID": 0,
 				"Activebool": false,
 				"CreateDate": "0001-01-01T00:00:00Z",
 				"LastUpdate": null,
 				"Active": null,
 				"Address": {
 					"AddressID": 502,
 					"Address": "1515 Korla Way",
 					"Address2": null,
 					"District": "",
 					"CityID": 0,
 					"PostalCode": null,
 					"Phone": "",
 					"LastUpdate": "0001-01-01T00:00:00Z"
 				}
 			}
 		]
 	}
 ]
 ```

All the fields missing source column in result set are initialized with empty value. 
City of `London` has two customers, which is the product of object reuse in `ROW 2` processing. 
 
### Custom model types

Destinations are not limited to just generated model types, any destination will work, as long as projection name
corresponds to `model type name`.`field name`. Only letters are compared, and cases(lowercase, uppercase, CamelCase,...) are ignored.  
**Go struct field has to be public for scan to work.**  
Field type can be of any base Go lang type, plus any type that implements `sql.Scanner` interface (UUID, decimal.Decimal{}, ...).
 
Lets rewrite above example to use custom model types instead generated ones:

```go
// Address struct has the same name and fields as auto-generated model struct
type Address struct {
    ID  	 int32 `sql:"primary_key"`
    AddressLine  string
}

type MyCustomer struct {
    ID         int32 `sql:"primary_key"`
    LastName   *string

    Address Address
}

type MyCity struct {
    ID     int32 `sql:"primary_key"`
    Name   string

    Customers []MyCustomer
}

dest2 := []MyCity{}

stmt2 := 
    SELECT(
        City.CityID.AS("my_city.id"),                 // snake case
        City.City.AS("myCity.Name"),                  // camel case
        Address.AddressID,                            // No need for aliasing. 
        Address.Address,                              // Default aliasing still works.  
        Customer.CustomerID.AS("My_Customer.id"),      //mixed case
        Customer.LastName.AS("my customer.last name"), //with spaces
    ).FROM(
        City.
            INNER_JOIN(Address, Address.CityID.EQ(City.CityID)).
            INNER_JOIN(Customer, Customer.AddressID.EQ(Address.AddressID)),
    ).WHERE(
        City.City.EQ(String("London")).OR(City.City.EQ(String("York"))),
    ).ORDER_BY(
        City.CityID, Address.AddressID, Customer.CustomerID,
    )

err := stmt2.Query(db, &dest2)
```

Destination type names and field names are now changed. Every type has 'My' prefix, every primary key column is named `ID`,
 `LastName` is now string pointer, etc.  
Now, since we use custom types with changed field identifiers, each column must have an alias for the query mapping to work.  
For instance: `City.CityID.AS("my_city.id")` -> `MyCity.ID`, `City.City.AS("myCity.Name")` -> `MyCity.Name` , etc.  

**Table names, column names and aliases doesn't have to be in a snake case. CamelCase, PascalCase or some other mixed space is also supported,
but it is strongly recommended to use snake case for database identifiers.**

Json of new destination is also changed:

```js
[
	{
		"ID": 312,
		"Name": "London",
		"Customers": [
			{
				"ID": 252,
				"LastName": "Hoffman",
				"Address": {
					"ID": 256,
					"AddressLine": "1497 Yuzhou Drive"
				}
			},
			{
				"ID": 512,
				"LastName": "Vines",
				"Address": {
					"ID": 517,
					"AddressLine": "548 Uruapan Street"
				}
			}
		]
	},
	{
		"ID": 589,
		"Name": "York",
		"Customers": [
			{
				"ID": 497,
				"LastName": "Sledge",
				"Address": {
					"ID": 502,
					"AddressLine": "1515 Korla Way"
				}
			}
		]
	}
]
```

#### Anonymous custom types

There is no need to create new named type every time. 
The destination type can be declared inline without naming of any new type.

```go
var dest []struct {
    CityID int32 `sql:"primary_key"`
    CityName   string

    Customers []struct {
        CustomerID int32 `sql:"primary_key"`
        LastName   string

        Address struct {
            AddressID   int32 `sql:"primary_key"`
            AddressLine string
        }
    }
}

stmt := 
    SELECT(
        City.CityID.AS("city_id"),
        City.City.AS("city_name"),
        Customer.CustomerID.AS("customer_id"),
        Customer.LastName.AS("last_name"),
        Address.AddressID.AS("address_id"),
        Address.Address.AS("address_line"),
    ).FROM(
        City.
            INNER_JOIN(Address, Address.CityID.EQ(City.CityID)).
            INNER_JOIN(Customer, Customer.AddressID.EQ(Address.AddressID)).
    )
    WHERE(City.City.EQ(String("London")).OR(City.City.EQ(String("York")))).
    ORDER_BY(City.CityID, Address.AddressID, Customer.CustomerID)

err := stmt.Query(db, &dest)
```
Aliasing is now simplified. Alias contains only (column/field) name. 
On the other hand, we can not have 3 fields named `ID`, because aliases must be unique.

#### Tagging model type fields

Desired mapping can be set the other way around as well, by tagging destination fields and types.

```go
var dest []struct {
    CityID   int32 `sql:"primary_key" alias:"city.city_id"`
    CityName string `alias:"city.city"`

    Customers []struct {
        // because the whole struct is refering to 'customer.*' (see below tag),
        // we can just use 'alias:"customer_id"`' instead of 'alias:"customer.customer_id"`'
        CustomerID int32 `sql:"primary_key" alias:"customer_id"` 
        LastName   *string `alias:"last_name"`                   

        Address struct {
            AddressID   int32 `sql:"primary_key" alias:"AddressId"` // camel case for alias will work as well
            AddressLine string `alias:"address.address"`            // full alias will work as well
        } `alias:"address.*"`                                       // struct is now refering to all address.* columns

    } `alias:"customer.*"`                                          // struct is now refering to all  customer.* columns
}

stmt := 
    SELECT(
        City.CityID,
        City.City,
        Customer.CustomerID,
        Customer.LastName,
        Address.AddressID,
        Address.Address,
    ).FROM(
        City.
            INNER_JOIN(Address, Address.CityID.EQ(City.CityID)).
            INNER_JOIN(Customer, Customer.AddressID.EQ(Address.AddressID)).
    ).
    WHERE(City.City.EQ(String("London")).OR(City.City.EQ(String("York")))).
    ORDER_BY(City.CityID, Address.AddressID, Customer.CustomerID)

err := stmt.Query(db, &dest)
```

This kind of mapping is more complicated than in previous examples, and it should avoided and used 
only when there is no alternative. Usually this is the case in two scenarios:

##### 1) Self join

```go
var dest []struct{
    model.Employee

    Manager *model.Employee `alias:"Manager.*"` //or just `alias:"Manager"
}

manager := Employee.AS("Manager")

stmt := 
    SELECT(
        Employee.EmployeeId,
        Employee.FirstName,
        manager.EmployeeId,
        manager.FirstName,
    ).FROM(
        Employee.
          LEFT_JOIN(manager, Employee.ReportsTo.EQ(manager.EmployeeId)).
    )

```
_This example could also be written without tag alias, by just introducing of a new type `type Manager model.Employee`._

##### 2) Slices of go base types (int32, float64, string, ...)

```go
var dest struct {
    model.Film
    
    InventoryIDs []int32 `alias:"inventory.inventory_id"`
}
```

### Combining autogenerated and custom model types

It is allowed to combine autogenerated and custom model types. 
For instance:

```go
type MyCustomer struct {
    ID         int32 `sql:"primary_key"`
    LastName   string

    Address    model.Address                  //model.Address is autogenerated model type
}

type MyCity struct {
    ID     int32 `sql:"primary_key"`
    Name   string

    Customers []MyCustomer
}
```
### Specifying primary keys

Model types generated from database views does not contain any field with `primary_key` tag. Because there is no `primary_key` field those types can not be used as a grouping destination. For instance:

```go
var dest []struct {
    model.ActorInfo       // <- view model file, without `primary_key` fields
    Films []model.Film    
}
```
Querying into above destination would not give correct result, because `Films` slice does not know to which `ActorInfo` it relates.  
To overcome this issue, we have to manually specify primary keys for view model types.

```go
var dest []struct {                               // ID is a field name in model.ActorInfo
    model.ActorInfo     `sql:"primary_key=ID"`    // coma separated list of field names
    Films []model.Film    
}
```
Above tag can be used to set new primary key fields on a model type with already defined primary key fields.
