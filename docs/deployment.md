## Self-Registration (Signup)

fan-files allows you to enable user self-registration (signup). This can be enabled via **Settings → Global Settings**, or with `fan-files config set --signup`. Self-registered users inherit the configured **user defaults**, including the scope.

> [!WARNING]
>
> By default, the user scope is the server's root, so a self-registered user could read,
> modify, and delete every file fan-files serves. To prevent this, either:
>
> a. Enable `createUserDir` so each user gets their own directory; or
> b. If users are meant to share files, set the default scope to something other than the root.

## Fail2ban

fan-files does not natively support protection against brute force attacks. Therefore, we suggest using something like [fail2ban](https://github.com/fail2ban/fail2ban), which takes care of that by tracking the logs of your fan-files instance. For more information on how fail2ban works, please refer to their [wiki](https://github.com/fail2ban/fail2ban/wiki).

### Filter Configuration

An example filter configuration targeted at matching fan-files's logs.

```ini
[INCLUDES]
before = common.conf

[Definition]
datepattern = `^%%Y\/%%m\/%%d %%H:%%M:%%S`
failregex   = `\/api\/login: 403 <HOST> *`
```

### Jail Configuration

An example jail configuration. You should fill it with the path of the logs of fan-files, as well as the port where it is running at.

```ini
[fan-files]

enabled = true
port = [your_port]
filter = fan-files
logpath = [your_log_path]
maxretry = 10
bantime = 10m
findtime = 10m
banaction = iptables-allports
banaction_allports = iptables-allports
```
