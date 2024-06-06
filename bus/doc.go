/*
  Package bus implements a thin wrapper around nsq so that you can create publishers for topics and
  consumers for channels.

  Publisher/Consumer

  `Consumer` and `Publisher` are thin wrappers around the basic concept of
  nsq with small additions.

  Functions

  A `Function` abstracts an asynchronous function which will be called by marshalling the
  parameters as JSON and invoke the corresponding function with nsq. You can start many services
  which implement the same function (identified by its name), so you will have something like a
  balancing. Note: As the functions are asynchronuous, you cannot return a result. Only errors
  are used to signal if the function was successful.

  When you create a function with `Unique` the consumer will be connected to a unique, ephemeral
  topic and channel. Create unique functions if you want your services to respond with values. You
  create a unique function, transport this function name to the wellknown service and this service
  will call the unique function with the result.

  If the process with the unique function ends, the topic and the channels will be removed from nsq
  because they are ephemeral.

  Creating a named function

     ep := NewEndpoints(...)
     f, err := ep.Function("hello-service", func (s string) error {
        fmt.Printf("Hello %s\n", s)
        return nil
     })
     f("world"); // prints "Hello world"

  creates a client and server for the function, depending on the endpoint. If there is a consumer in
  the endpoint, a server is registered; if there is a publisher a client is also created. So using an
  endpoint with consumer and publisher creates a function which will be hosted by the same process which
  creates the function.

  When you only need a client for a wellknown function, use the `Client` function:

    ep := NewEndpoints(...)
    f, err := ep.Client("hello-service")
    f("world")

  In this case there should be another process which registered a consuming function.

  Last but not least you can use `Unique` to create a unique consumer which only exists as long
  as the process exists which did the registration. Unique consumers return their name and they
  can be used to transport responses from well known services back to clients. The client has
  to register a unique consumer and pass the name of this function to the service which will post
  back the response back to the client.
*/
package bus
