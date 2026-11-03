# First admin

A new authority has no users, so nobody can sign in. Create the first one with a local password:

```sh
authority create-initial-admin alice
```

It asks for the password twice without showing it. The password needs at least 15 characters and at most 1024 bytes. The authority stores only its argon2id hash. Before hashing, it normalizes the password with Unicode NFKC, so a character like é matches however a keyboard or system encodes it.

If stdin isn't a terminal, the command reads the password once, from the first line of stdin. Use this in a script, and keep the password out of arguments and environment variables, where other processes can read it:

```sh
authority create-initial-admin alice < password.txt
```

Note that stdin is a terminal only with `kubectl exec -it` or `ssh -t`. Without `-t`, the command shows no prompt, and your terminal shows the password as you type it. Without `-i`, `kubectl exec` passes no stdin at all.

The command uses the same `TIDEGATE_AUTHORITY__DATABASE__URL` as the authority, and the database must be migrated first. It works only while the database holds no user. Once a user exists, it fails and changes nothing, also when two runs start at the same time.
