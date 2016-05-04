## Making a dex Release

1. Make a new clone of the dex repo:

  ```console
  $ git clone git@github.com:coreos/dex.git
  $ cd dex
  ```

1. Tag with the release name:

   ```console
   git tag -s v0.4.0
   ```

1. Push the change:
    ```console
    git push origin v0.4.0
    ```

1. Make a release with release notes here:
    https://github.com/coreos/dex/releases
    
