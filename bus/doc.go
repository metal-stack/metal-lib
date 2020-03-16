/*
  Package bus implements a thin wrapper around nsq so that you can create publishers for topics and
  consumers for channels.

  `Consumer` and `Publisher` are thin wrappers around the basic concept of
  nsq with small additions.

  A `Function` abstracts an asynchronous function which will be called by marshalling the
  parameters as JSON and invoke the corresponding function with nsq. You can start many services
  which implement the same function (identified by its name), so you will have something like a
  balancing. Note: As the functions are asynchronuous, you cannot return a result. Only errors
  are used to signal if the function was successfull.
*/
package bus
