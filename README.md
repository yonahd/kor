# Kor - Kubernetes Orphaned Resources Finder

Kor is a CLI tool to discover unused Kubernetes resources. Currently, Kor can identify and list unused ConfigMaps and Secrets.

![Kor Screenshot](/images/screenshot.png)

## Installation

Download the binary for your operating system from the [releases page](https://github.com/yonahd/kor/releases) and add it to your system's PATH.

## Usage

Kor provides various subcommands to identify and list unused resources. The available commands are:

- `all`: Gets all unused resources (configmaps, secrets, services, and service accounts) for the specified namespace or all namespaces.
- `configmap`: Gets unused configmaps for the specified namespace or all namespaces.
- `secret`: Gets unused secrets for the specified namespace or all namespaces.
- `services`: Gets unused services for the specified namespace or all namespaces.
- `serviceaccount`: Gets unused service accounts for the specified namespace or all namespaces.

To use a specific subcommand, run `kor [subcommand] [flags]`.

```sh
kor all --namespace my-namespace
```

For more information about each subcommand and its available flags, you can use the `--help` flag.

```sh
kor [subcommand] --help
```

## Contributing

Contributions are welcome! If you encounter any bugs or have suggestions for improvements, please open an issue in the [issue tracker](https://github.com/yonahd/kor/issues).

## License

This project is open-source and available under the [MIT License](LICENSE). Feel free to use, modify, and distribute it as per the terms of the license.