# Security Policy

## Supported Versions

We release patches for security vulnerabilities in the following versions:

| Version | Supported          |
| ------- | ------------------ |
| main    | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue, please follow these steps:

### 1. Do Not Open a Public Issue

Please **do not** create a public GitHub issue for security vulnerabilities. Public disclosure could put users at risk.

### 2. Report Privately

Send details to: **[security contact - update with actual email]**

Include in your report:
- Description of the vulnerability
- Steps to reproduce the issue
- Potential impact
- Suggested fix (if any)

### 3. Response Timeline

- **Initial response**: Within 48 hours
- **Status update**: Within 5 business days
- **Fix timeline**: Depends on severity
  - Critical: Within 7 days
  - High: Within 14 days
  - Medium: Within 30 days
  - Low: Next regular release

### 4. Disclosure Process

1. We'll acknowledge receipt of your vulnerability report
2. We'll investigate and validate the issue
3. We'll develop and test a fix
4. We'll release the fix in a new version
5. We'll publicly disclose the vulnerability (with credit to reporter if desired)

## Security Considerations

### Local Data Storage

diaryctl stores diary entries locally:
- **Markdown backend**: Plain text files in `~/.diaryctl/data/`
- **SQLite backend**: Database file in `~/.diaryctl/data/diary.db`

**Important**: These files are not encrypted. Do not store highly sensitive information without additional encryption at the filesystem level.

### Configuration Security

Configuration file: `~/.config/diaryctl/config.yaml`

- Uses standard file permissions (user read/write only)
- Does not store credentials or API keys
- Safe to commit example configs (without personal data)

### Command Injection Risks

diaryctl uses `exec.Command` for:
- Editor invocation (via `$EDITOR` env var)
- Git operations (hardcoded git commands only)

User input is **not** passed directly to shell commands. All command execution uses parameterized arguments.

### Dependencies

We regularly update dependencies to address security vulnerabilities. Dependency security is monitored via:
- GitHub Dependabot
- Go vulnerability database (`govulncheck`)

### Turso/SQLite Remote Sync

If using Turso for remote database sync:
- Connections use HTTPS/WSS encryption
- Authentication tokens should be stored securely
- Tokens should have minimal necessary permissions

## Security Best Practices for Users

1. **File Permissions**: Ensure data directory has restricted permissions:
   ```bash
   chmod 700 ~/.diaryctl/data
   ```

2. **Backups**: Regularly backup your diary data
   ```bash
   tar -czf diary-backup.tar.gz ~/.diaryctl/data
   ```

3. **Sensitive Data**: Consider full-disk encryption for highly sensitive diary entries

4. **Editor Security**: Use a trusted text editor via `$EDITOR` environment variable

5. **Remote Sync**: If using Turso, rotate access tokens periodically

## Known Security Limitations

- No built-in encryption for stored diary entries
- No authentication/authorization (single-user CLI tool)
- Editor invocation trusts `$EDITOR` environment variable

## Security Updates

Security fixes will be announced via:
- GitHub Security Advisories
- Release notes
- README.md security section

## Questions?

For security-related questions (not vulnerabilities), open a GitHub Discussion or issue.
