package muxy

/*
Package gorilla/muxy is a request router designed for easy of use, power and
speed. The main features are:

	- Extensive URL matching: match scheme, host and path using variables in
	  a single pattern syntax.
	- Middleware support: register middleware handlers to be applied on a
	  matched URL.
	- Reversible routes: build URLs for a named route using the provided
	  variables.
	- Subrouters: create convenience routers that share middleware or
	  pattern and name prefixes.
	- Slim, incremental API: get what you need now and come back later for more.
	  The interface is carefully focused on intuitiveness and easy of use for
	  the common cases while still providing unmatchable features for
	  advanced scenarios.
	- High performance: enjoy the ride! Best-of-breed matching speed without
	  compromising easy of use. No API weirdness for a few nanoseconds, man.

Let's start with the most simple use case:

	r := muxy.New()
	r.Route("/foo").Get(ShowFoo).Post(SaveFoo)

Here we created a new router and registered a route for the path "/foo". Then
we defined a handler to be served for the GET request method, and another one
for the POST method.

...

*/
