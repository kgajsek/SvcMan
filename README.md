# SvcMan - Poor man's service manager

This is a very basic service manager. It is used in production, however, only for legacy ASP.NET applications, where I needed to add some microservices and other solutions were either overkill or consumed too much resources in limited or restricted environments. Use it if your use case fits such scenarios, otherwise switch to more renowned service managers or go with containers, serverless, etc...

# How it works

SvcMan runs on port 9500. It handles requests in form:

```
http://localhost:9500/serviceName/param1/param2/...
```

When the first request for a particular service comes in, SvcMan will assign it a port (9600-9699) and start an executable with serviceName and its port and then forward that request to it. All further requests are just forwarded.

Services should be placed in sub directories under SvcMan. Each service can have many versions, SvcMan will just execute the latest (ordered by directory name).

Example:

With directory structure:

```
  SvcMan
    Service1
      2021-03-01
        Service1.exe
      2021-03-18
        Service1.exe
    Service2
      2021-03-15
        Service2.exe
```

a request to: 

```
http://localhost:9500/Service1/p1/p2
```

SvcMan will start Service1\2021-03-18\Service1.exe and pass it its port (9600 if this is the first service) as parameter, then forward the request to

```
http://localhost:9600/p1/p2
```

# Service requirements

Each service should expose the following endpoints, to be SvcMan compatible:

```
/echo - it should return "OK"
/stop - it should stop the service (may complete any opened tasks, etc.)
```

and it must be compiled as executable and bind themselves to port given as first command line parameter.

# Upgrading services

Services can be upgraded on the fly. Simply create a new version directory, copy new version of your service into it, and issue SvcMan stop command for that service, e.g.:

```
http://localhost:9500/stop?svc=Service1
```

SvcMan will stop the currently running version of the service, and start a new version as soon as next request for that service arrives.

It is also possible to stop all running services with:

```
http://localhost:9500/stop?svc=all
```

# Calling other services

Services can call other services the same way, and do not need to know their ports:

```
http://localhost:9500/otherService/param1/param2/...
```

