# GoBack
[![Coverage](https://codecov.io/gh/vanvanni/goback/branch/develop/graph/badge.svg)](https://codecov.io/gh/vanvanni/goback)
[![Go Report Card](https://goreportcard.com/badge/github.com/vanvanni/goback)](https://goreportcard.com/report/github.com/vanvanni/goback)
[![Go Version](https://img.shields.io/github/go-mod/go-version/vanvanni/goback)](https://github.com/vanvanni/goback/blob/develop/go.mod)
[![Stars](https://img.shields.io/github/stars/vanvanni/goback?style=social)](https://github.com/vanvanni/goback/stargazers)
[![Forks](https://img.shields.io/github/forks/vanvanni/goback?style=social)](https://github.com/vanvanni/goback/network/members)
[![License](https://img.shields.io/github/license/vanvanni/goback)](https://github.com/vanvanni/goback/blob/develop/LICENSE)
[![Last Commit](https://img.shields.io/github/last-commit/vanvanni/goback)](https://github.com/vanvanni/goback/commits/develop)

GoBack was created out of the need for a simple, minimal, and easy-to-use backup solution. While many existing backup software options are powerful, they were often too complex to set up or didn't cover specific needs. GoBack aims to be a straightforward tool that can run as a system service and reliably do its job.

It supports scheduling via Crontab expressions and intervals, allowing for flexible backup intervals, such as every 57 seconds.

**Current Status:** This project is a Work In Progress (W.I.P). Features are being developed and tested.

**Disclaimer:** Most of the tests in this project were written with AI assistance, because the generated tests are generally better than the ones I write manually.

## Features

*   **Database Backups:** Support for MariaDB.
*   **Volume Backups:** Backup Docker volumes and other mounted storage.
*   **Directory Backups:** Simple file and directory backups.
*   **Cloud Storage Integration:** Store backups on S3 compatible storage.
*   **Compression:** Reduce backup size for efficient storage.
*   **Encryption / Decryption:** AES-GCM encryption during backup and CLI decryption support.
*   **Flexible Scheduling:** Use Crontab expressions or intervals for scheduling.
*   **CLI Commands:** Run the service with `goback start` and decrypt archives with `goback decrypt`.

## Roadmap

The following features are planned for future releases:

*   **PostgreSQL Support**
*   **MongoDB Support**
*   **MySQL Support** (Currently use the MariaDB driver)

## Installation (W.I.P)

Installation instructions will be provided once the project reaches a more stable state.

### Automated Installation

### Manual Installation

## Usage

Start GoBack:

```bash
goback start --dir /etc/goback
```

Decrypt an encrypted archive:

```bash
goback decrypt /path/to/backup.tar.gz.enc --key "your-key"
```

Or resolve key from config/spec:

```bash
goback decrypt /path/to/backup.tar.gz.enc my-spec.yml --dir /etc/goback
```

## License

This project is licensed under the GNU General **Public License v3.0** License - see the [LICENSE](LICENSE) file for details.
