# taylor
distributed worker queue written in golang

## Dependencies

- uuid
- gin
- ql

## To-Do

- consul integration
- docker driver
- automatic socket updates
- add job.OnError {reschedule, webhook callback, script callback, nothing}
- add job.OnSuccess {webhook callback, script callback, nothing}
- add TLS
