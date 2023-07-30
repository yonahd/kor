# Kor - Kubernetes Orphaned Resources Finder

Kor is a CLI tool to discover unused Kubernetes resources. Currently, Kor can identify and list unused:
- ConfigMaps  
- Secrets.
- Services
- ServiceAccounts
- Deployments
- Statefulsets
- Roles

![Kor Screenshot](/images/screenshot.png)

## Installation

Download the binary for your operating system from the [releases page](https://github.com/yonahd/kor/releases) and add it to your system's PATH.

## Usage

Kor provides various subcommands to identify and list unused resources. The available commands are:

- `all`: Gets all unused resources for the specified namespace or all namespaces.
- `configmap`: Gets unused configmaps for the specified namespace or all namespaces.
- `secret`: Gets unused secrets for the specified namespace or all namespaces.
- `services`: Gets unused services for the specified namespace or all namespaces.
- `serviceaccount`: Gets unused service accounts for the specified namespace or all namespaces.
- `deployments`: Gets unused service accounts for the specified namespace or all namespaces.
- `statefulsets`: Gets unused service accounts for the specified namespace or all namespaces.
- `role`: Gets unused roles for the specified namespace or all namespaces.

To use a specific subcommand, run `kor [subcommand] [flags]`.

```sh
kor all --namespace my-namespace
```

For more information about each subcommand and its available flags, you can use the `--help` flag.

```sh
kor [subcommand] --help
```

## Supported resources and limitations

| Resource        | What it looks for                                                                                                                                                                                                                  | Known False Positives                                                                                                        |
|-----------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------|
| Configmaps      | Configmaps not used in the following places:<br/>- Pods<br/>- Containers <br/>- Configmaps used through volumes <br/>- Configmaps used through environment variables                                                               | Configmaps used by resources which don't explicitly state them in the config.<br/> e.g Grafana dashboards loaded dynamically |
| Secrets         | Secrets not used in the following places:<br/>- Pods<br/>- Containers <br/>- Secrets used through volumes <br/>- Secrets used through environment variables<br/>- Secrets used by ingress TLS<br/>-Secrets used by ServiceAccounts |                                                                                                                              |
| Services        | Services with no endpoints                                                                                                                                                                                                         |                                                                                                                              |
| Deployments     | Deployments with 0 Replicas                                                                                                                                                                                                        |                                                                                                                              |
| ServiceAccounts | ServiceAccounts unused by pods<br/>ServiceAccounts unused by roleBinding or clusterRoleBinding                                                                                                                                     |                                                                                                                              |
| Statefulsets    | Statefulsets with 0 Replicas                                                                                                                                                                                                     |                                                                                                                              |
| Roles           | Roles not used in roleBinding                                                                                                                                                                                                      |                                                                                                                              |



## Contributing

Contributions are welcome! If you encounter any bugs or have suggestions for improvements, please open an issue in the [issue tracker](https://github.com/yonahd/kor/issues).

## License

This project is open-source and available under the [MIT License](LICENSE). Feel free to use, modify, and distribute it as per the terms of the license.

