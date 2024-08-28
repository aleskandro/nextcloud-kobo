# Nextcloud Sync Daemon for Kobo eReaders

![Go Version](https://img.shields.io/badge/go-1.23%2B-blue)
![License](https://img.shields.io/github/license/aleskandro/nextcloud-kobo?)
![Build Status](https://github.com/aleskandro/nextcloud-kobo/actions/workflows/build.yaml/badge.svg)
![GitHub Downloads (all assets, latest release)](https://img.shields.io/github/downloads/aleskandro/nextcloud-kobo/latest/total?)

## Overview

**Nextcloud Sync Daemon for Kobo** is a Golang-based software designed to run on Kobo eReaders, allowing users to
synchronize a list of Nextcloud remote endpoints and folders back to a folder on the Kobo filesystem. This daemon is
automatically activated every time the Kobo device connects to the internet, ensuring that your files are always
up-to-date.

Note: This software has been tested only on the *Kobo Elipsa 2e*. While it may work on other Kobo devices, compatibility
is not guaranteed.

## Features

- **Automatic Synchronization**: Syncs specified folders from Nextcloud to a designated folder on your Kobo eReader
  every time it connects to the internet.
- **Support for Multiple Remotes**: Manage and sync multiple Nextcloud endpoints and folders.
- **Daemon Mode**: Runs quietly in the background as a daemon process.
- **Efficient Syncing**: Downloads only updated or new files to minimize data usage and speed up synchronization.

## Installation

To install the Nextcloud Sync Daemon on your Kobo eReader, follow these steps:

### Prerequisites

### Steps

1. **Download the KoboRoot.tgz**: Go to the [releases page](https://github.com/yourusername/kobo-nextcloud-sync/releases) and
   download the latest release with your Kobo device.

2. **Transfer the binary to your Kobo**: Connect your Kobo eReader to your computer via USB and copy the downloaded
   file to the Kobo's internal storage at `(/mnt/onboard).kobo/KoboRoot.tgz`.

3. **Configure the daemon**:
   The daemon reads the configuration from `(/mnt/onboard).adds/nextcloud-kobo/config.yaml`.
   Here is an example configuration:

```yaml
autoUpdate: true # Automatically update the daemon from the GitHub release page
remotes:
- url: https://nextcloud.jdoe.com/s/abc123
  local_path: share1/
- url: https://nextcloud.jdoe.com/
  username: john # Do not set if using a share link
  password: doe
  remote_folder: /my-remote-folder/ # Do not set if using a share link
  local_path: share2/
- url: https://nextcloud.jdoe.com/s/abc123
  password: doe
  local_path: share3/
```

4. **Reboot your Kobo**: Safely eject your Kobo device from your computer and reboot it to apply the changes.

5. If the configuration is correct, you will get a message in the UI when the synchronization is complete.

## Usage

Once installed and configured, the Nextcloud Sync Daemon will automatically sync the specified folders every time your
Kobo eReader connects to the internet.

### Logs

Logs are generated in the `/mnt/onboard/.adds/nextcloud-kobo/nextcloud-kobo.log` directory on your Kobo device. 

## Configuration

The `config.yaml` file is the core configuration file for this daemon.

### Configuration Options

- **auto_update**: If set to `true`, the daemon will automatically update from the GitHub release page after the first run.
- **repo_owner**: defaults to `aleskandro` and used as the source for the repo owner of the automatic updates (override if forking).
- **repo_name**: defaults to `nextcloud-kobo` and used as the source for the repo name of the automatic updates (override if forking).
- **remotes**: a list of Nextcloud remotes to sync with the Kobo device.

#### Remote Options

- **URL**: The Nextcloud share link for the folder you want to sync or the nextcloud URL for user-password authentication.
- **userName**: Your Nextcloud username. Leave empty if you are using a share link.
- **password**: Your Nextcloud password or the share link password.
- **remoteFolder**: The folder on the Nextcloud server that you want to sync. Leave empty if you are using a share link.
- **localPath**: The path on your Kobo device where the files will be synchronized. It is a relative path that will be
 created in the `/mnt/onboard/nextcloud` directory.

## Contributing

We welcome contributions! To contribute to the project:

1. Fork the repository.
2. Create a new branch for your feature or bugfix.
3. Commit your changes and push them to your fork.
4. Open a pull request detailing your changes.

Please make sure to update tests as appropriate and adhere to the code style.

### Requirements

- [Docker](https://docs.docker.com/get-docker/) or [Podman](https://podman.io/getting-started/installation)
- [Go](https://golang.org/doc/install)
- [Make](https://www.gnu.org/software/make/)

### Running Tests

To run the tests, execute the following command:

```bash
make static
make test
```

### Building

To build the project, execute the following command:

```bash
make koboroot
```

The output KoboRoot.tgz file will be located in the `_artifacts` directory.

## License

This project is licensed under the Apache License - see the [LICENSE](LICENSE) file for details.

## Support

If you encounter any issues or have questions, please open an issue on this GitHub repository.
