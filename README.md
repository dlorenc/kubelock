# Kubelock

Kubelock is a simple Kubernetes-based CLI for waiting, acquiring and holding cluster-wide locks.

## Usage

Get a lock and hold it until ctrl-c:

```shell
$ kubelock --hold
dlorenc-macbookpro:kubelock dlorenc$ ./main --hold
Checking that lock exists...
Trying to get lock...
Got lock!
refreshing lock...
```

The same command waits for the lock to be free:

```shell
$ kubelock
Checking that lock exists...
Trying to get lock...
error getting lock: dlorenc-macbookpro.roam.corp.google.com already holds the lock for 7.441934087 seconds
retrying
Trying to get lock...
error getting lock: dlorenc-macbookpro.roam.corp.google.com already holds the lock for 2.439540372 seconds
retrying
Trying to get lock...
Got lock!
```

## Lock Details

Locks are based on `Annotations` on a `ConfigMap`.
Locks are scoped to a Kubernetes namespace.

The program creates a `ConfigMap` in the specified namespace named `kubelock`, with the following annotations:

### expiry

The UTC-based Unix timestamp that the lock should expire at.
This is currently hardcoded to be 60 seconds after the time the lock is acquired.

### holder

The name of the current program holding the lock. This defaults to the machine hostname but can be overriden with the -clientName flag.
