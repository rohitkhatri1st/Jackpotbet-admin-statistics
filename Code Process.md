I created the assignment md first.

Init git

Init go mod

Created initial structure with empty folders
wrote main.go
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
Made the structure easily swappable keeping in mind about easy scaling or easily changing infra. However out of scope of this project but necessary
