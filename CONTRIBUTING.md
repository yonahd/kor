## Contributing to kor

Thank you for considering contributing to `kor`! We appreciate your interest in helping us improve our project and push the K8s community forward.

### How Can I Contribute?

There are several ways you can contribute to `kor`:

1. **Reporting Bugs**: If you encounter a bug or unexpected behavior, please open an issue on our GitHub repository. Please include as much detail as possible to help us reproduce and fix the issue.

2. **Suggesting Enhancements**: If you have ideas for new features or improvements, feel free to suggest them by opening an issue. We welcome your input and feedback!

3. **Submitting Pull Requests**: If you'd like to contribute directly to the codebase, you can submit pull requests with bug fixes, new features, or improvements. Please make sure to follow our coding guidelines and provide clear explanations for your changes.

### Join Our Community

We have a Discord server where you can engage with other contributors, ask questions, and discuss ideas related to `kor`. Join us here to connect with the community!

[![Discord](https://discord.com/api/guilds/1159544275722321990/embed.png)](https://discord.gg/ajptYPwcJY)

### Getting Started

To get started contributing, follow these steps:

1. Fork the repository on GitHub.
2. Clone your forked repository to your local machine.
3. Install any necessary dependencies.
4. Make your changes and test thoroughly.
5. Commit your changes with clear and descriptive commit messages.
6. Push your changes to your fork on GitHub.
7. Submit a pull request to the main repository, clearly explaining the changes you've made.
8. If the PR is related to an open issue, link it. Successfullly merging might close it.

### Repository Structure
As adding new orphaned resources capabilities requires the addition or modification of multiple files in the repo, here are some highlighted files to simplify the process:

```
.
├── charts/kor/templates
│   └── role.yaml
├── cmd/kor
│   └── <resource>s.go
├── pkg/kor
│   ├── all.go
│   ├── create_test_resources.go
│   ├── delete.go
│   ├── multi.go
│   ├── <resource>s.go
│   └── <resource>s_test.go
└── README.md
```

- `pkg/kor/<resource>s.go` - add a new capability to map and manage unused objects of type \<resource>.
- `pkg/kor/<resource>s_test.go` - add a Go test suite to cover your new methods.
- `pkg/kor/create_test_resources.go` - create a test resource of type \<resource>.
- `pkg/kor/all.go` - add your new resource to `kor all` command to map all unused resources.
- `pkg/kor/delete.go` - add a deletion functionality to apply on unused instances of type \<resource>.
- `pkg/kor/multi.go` - allow finding your new resource in a comma-separated query along other resources.
- `cmd/kor/<resource>s.go` - add your new functionanilities to `kor` command-line.
- `charts/kor/templates/role.yaml` - grant get/list/watch permissions to the new resource in a namespaces/cluster-scoped level.
- `README.md` - introduce your added capabilities to `kor`.

### Code of Conduct

Our project follows the [MIT License](https://github.com/yonahd/kor/blob/main/LICENSE), allowing for open collaboration and distribution. We expect all participants to engage respectfully and professionally in discussions and contributions, ensuring a respectful and inclusive environment for everyone involved. Failure to comply may result in appropriate actions determined by project maintainers.

### Resources

- [Issue Tracker](https://github.com/yonahd/kor/issues): Report bugs and suggest enhancements.
- [Pull Requests](https://github.com/yonahd/kor/pulls): Submit pull requests with your contributions.

Thank you for contributing to `kor`! We look forward to your involvement in our project. If you have any questions, feel free to reach out to us.
