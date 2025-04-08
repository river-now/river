1. bump package.json / run build / publish to npm

```sh
make npmbump
```

2. push to github

```sh
git commit -am 'v0.0.0-pre.0'
git push
```

3. publish to go proxy / push version tag

```sh
make gobump
```

4. profit
