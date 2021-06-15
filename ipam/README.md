# Das SCHIFF Engine Operator

The engine operator contains all custom controllers and resources that are used within Das SCHIFF. It's built using the [Operator SDK](https://sdk.operatorframework.io/).

## Building

To build you'll need to have a working Go development environment that's at least the version specified in the `./go.mod` file.
Then simply run `make`.

## Testing

The `envtest` package of https://github.com/kubernetes-sigs/controller-runtime is used for testing. It requres a set of tools to be located in `/usr/local/kubebuilder/bin`. To install them run the following command

```bash
sudo hack/setup-envtest.sh
```

You can then execute all tests by running:

```
make test
```
