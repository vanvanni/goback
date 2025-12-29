# GoBack (W.I.P)

GoBack was created out of the need for a simple, minimal, and easy-to-use backup solution. While many existing backup software options are powerful, they were often too complex to set up or didn't cover specific needs. GoBack aims to be a straightforward tool that can run as a system service and reliably do its job.

It supports scheduling via Crontab expressions and intervals, allowing for flexible backup intervals, such as every 57 seconds.

**Current Status:** This project is a Work In Progress (W.I.P). Features are being developed and tested.

## Features

*   **Database Backups:** Support for MariaDB.
*   **Volume Backups:** Backup Docker volumes and other mounted storage.
*   **Directory Backups:** Simple file and directory backups.
*   **Cloud Storage Integration:** Store backups on S3 compatible storage.
*   **Compression:** Reduce backup size for efficient storage.
*   **Flexible Scheduling:** Use Crontab expressions or intervals for scheduling.

## Roadmap

The following features are planned for future releases:

*   **Encryption:** Secure your backups with encryption.
*   **PostgreSQL Support**
*   **MongoDB Support**
*   **MySQL Support** (Currently use the MariaDB driver)
*   **S3 Uploads**
*   **CLI Commands**

## Installation (W.I.P)

Installation instructions will be provided once the project reaches a more stable state.

### Automated Installation

### Manual Installation

## Usage (W.I.P)

Usage examples and command-line interfaces will be documented as features mature.

## License

This project is licensed under the GNU General **Public License v3.0** License - see the [LICENSE](LICENSE) file for details.
