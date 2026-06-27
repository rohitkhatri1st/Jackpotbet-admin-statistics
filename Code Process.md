# Code Process

I created the assignment md first.

Init git

Init go mod

Created initial structure with empty folders

wrote main.go

Choosing 3-layer architecture Handler → Service → Repository, since we want http parsing, then business logic and then db aggregation. If it were only crud we would have eliminated business/service layer, and creating more layers like domain/adapters would be an overkill for our operations that generally require interaction only within the system. Events are not required in the current scope, we may introduce them separately when required.

Choosing repository pattern because we may want to create mongo pipelines and we must give those pipelines a place to sit.

Following SOLID Principles for least code complexity.

Created server package

Create logger

Structure made by keeping in mind EC2 instances

Init DBs

Init empty services

Added config to fetch required config

Chose viper over burntsushi or other toml parser as viper allows multipath file search in itself while in others we have to build it and we can easily name different environments. However burntsushi is stricter for unknown keys while viper is not, this can be traded off.

Chose mux over gin, iris or other frameworks as we want to use net/http types directly and have full control of things in our hands while other frameworks locks us into their own context and the requirement to use go-playground/validator for input validation means that we won't be using gin's validation so it leaves us with practically no advantage on using gin while it is locking us in.

Created all services as easily swappable interfaces

Checked if logger is easily swappable or not to ensure structural integrity

Made the entire interfaces and structure easily swappable and mockable for testing and keeping in mind about easy scaling or easily changing infra. However out of scope of this project but necessary

Now making routes

Making a create transaction route so that its code may be reused to easily create the utility script to fill data as per the requirement of assignment.

Making a better validator package with underlying playground validator for more control over it

Creating a decodeJsonBody function for better json decoding and error handling

Creating methods that can respond to the apis in a structured and controlled way.

Creating APIs for creating transactions to test the strucutre and then maybe reuse its code to write the utility script to add transactions data as well as per the assignments requirement

Created getTransactions api to create a particular pattern that can be followed throughout and to test if creation api worked fine or not.

Creating a separate date filter schema so that every api can use it whenever required

Creating bulk insert repository layer to use it in the utility script that fills the db

Creating the utility script.

Creating stages in the utility script so that its easily manageable and understandable.

Creating required APIs

Creating gross_gaming_rev api

Creating an ensure index code and minor code refactoring to support that
