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
  are used to signal if the function was successfull.

  When you create a function with `Unique` the consumer will be connected to a unique, ephemeral
  topic and channel. Create unique functions if you want your services to respond with values. You
  create a unique function, transport this function name to the wellknown service and this service
  will call the unique function with the result.

  If the process with the unique function ends, the topic and the channels will be removed from nsq
  because they are ephemeral.
*/
package bus
