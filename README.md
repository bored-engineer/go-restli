# go-restli: Golang bindings for Rest.li

## How to
Grab a binary from the latest [release](https://github.com/PapaCharlie/go-restli/releases) for your platform and
put it on your path. You can now use this tool to generate Rest.li bindings for any given resource. You will need to
acquire all the PDSC/PDL models that your resources depend on, as well as their restspecs. Once you do, you can run the
tool as follows:
```bash
go-restli \
  --package-prefix github.com/PapaCharlie/go-restli/tests/generated \
  --output-dir ./tests/generated \
  --schema-dir ./pegasus \
  idl/*.restspec.json
```
+ **--package-prefix**: All files will be generated inside of this namespace (e.g. `generated/`), and the generated
  code will need to be imported accordingly.
+ **--output-dir**: The directory in which to output the files. Any necessary subdirectories will be created.
+ **--schema-dir**: The directory that contains all the `.pdsc` and `.pdl` files that may be used by the resources you
  want to generate code for.
+ All remaining parameters are the paths to the restspec files for the resources you want to call.

### Note on Java dependency
The owners of Rest.li recommended against implementing a custom PDSC/PDL/RESTSPEC parser and instead recommend using
the existing Java code to parse everything. This is not only because the .pdsc format is going to be replaced by a new
DSL called PDL, which will be much harder to parse than JSON (incidentally, the .pdsc format allows comments and other
nonsense, which makes it not standard JSON either). Therefore this code actually uses Java to parse everything, then
outputs a simpler intermediary JSON file where every schema and spec is fully resolved, making the code generation step
significantly less complicated.

In order to parse the schemas and restspecs, the binaries have an embedded jar. They will unpack the jar and attempt to
execute it with `java -jar`. This jar has no dependencies, but you _must_ have a JRE installed. Please make sure that
`java` is on your PATH (though setting a correct `JAVA_HOME` isn't 100% necessary). This has been tested with Java 1.8.

## Getting the PDSCs and Restpecs
You may wish to use gradle to extract the schema and restspec from the incoming jars. To do so, you can use a task like
this:
```gradle
task extractPdscsAndRestpecs >> {
  copy {
    project.configurations.restliSpecs.each {
      from zipTree(it)
      include "idl/"
      include "pegasus/"
    }

    into temporaryDir
  }
}
```

## Conflict resolution in cyclic packages
Java allows cyclic package imports since multiple modules can define classes for the same packages. Similarly, it's
entirely possible for schemas to introduce package cycles. To mitigate this, the code generator will attempt to resolve
dependency chains that introduce package cycles and move the offending models to a fixed package called
`conflictResolution`.

## TODO
There are still many missing parts to this, including documentation and polish. I first focused on the biggest pain
point in working with Rest.li in golang, which is to generate the structs that are used to send and receive requests to
Rest.li endpoints. Most of the useful constants like resource paths and action names get extracted from the spec as
well, just to make it easier to write the code against net/http.Client, or whatever your favorite HTTP client framework
might be.

## Contributing to this project
First, you have to clone this repo and all its submodules:
```bash
% git clone --recurse-submodules git@github.com/PapaCharlie/go-restli
```
There exists a testing framework for Rest.li client implementations that provide expected requests and responses. The
[gorestli_test.go](tests/gorestli_test.go) and [manifest.go](tests/manifest.go) files read the testing
[manifest](tests/rest.li-test-suite/client-testsuite/manifest.json) and load all the corresponding requests and
responses. Once a new resource type or method is implemented, please be sure to integrate all the tests for that new
feature. This is done by adding a new function on the `TestServer` struct for the corresponding name. For example, to
test the `collection-get` test (see the corresponding object in the manifest), all that's needed is to add a
correspondingly named method called `CollectionGet`, like this:
```go
func (s *TestServer) CollectionGet(t *testing.T, c *Client) {
	id := int64(1)
	res, err := c.Get(id)
	require.NoError(t, err)
	require.Equal(t, &conflictresolution.Message{Id: &id, Message: "test message"}, res, "Invalid response from server")
}
```
Once you have written your tests, just run `make` in the root directory and all the tests will be run.
