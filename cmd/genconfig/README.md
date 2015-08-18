genconfig
====

The genconfig tool generates boilerplate code for registering and deserializing configuration objects.

## Usage - Go source

Let's say you have an interface called `Foo` with several concrete implmentations, and you'd like to be able to instantiate these implementations by reading JSON configuration files.

The first thing you do is create a new interface called `FooConfig` and add a `go:generate` directive to it like so:

    //go:generate genconfig -o config.go foo Foo
    type FooConfig interface {
      FooID() string
      Foo()
      ...

The first argument is the name of the file to create. The second (`foo`) is the name of the package. The third is the object that is to be configurable.

## Usage - Go Generate

To generate (or re-generate) your config file, issue the following command in your shell:
    ```
    go generate github.com/coreos/dex/foo
    ```

## Usage - Generated Code API

Every time you make a new `Foo` implementation, you should create a new `FooConfig` which configures it, and can return a `Foo`. You should tag `FooConfig` so that it can be serialized/de-serialized as you plesae.

After creating the `Config` object, you can register it with the the generated `RegisterFooConfigType`. This allows you to create `NewFooFromType(fooType string)` and also `newFooConfigFromMap`.

In practice you will do something like: deserialize a JSON object into a `map[string]interface{}`, pass that map into `newFooConfigFromMap` which gives you your `FooConfig` and then call whatever you've implemented to get a `Foo` from a `FooConfig`

## The Future?

There's still a lot of boilerplate that needs to be generated to use this API. It would be nice to generate even more code, with more `go:generate` directives. For example, partial implementations of the XXXConfig objects could be generated, along with functions for deserializing the Config objects (or slices of them) from an io.Reader.




