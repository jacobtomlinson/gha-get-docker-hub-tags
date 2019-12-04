# Get Docker Hub Tag

Get the latest tag for an image on Docker Hub.

## Usage

Add following step to the end of your workflow:

```yaml
    - name: Get latest tag
      id: latest_tag
      uses: jacobtomlinson/gha-get-docker-hub-tags
      with:
        org: 'mysql'  # Docker Hub user or organisation name
        repo: 'mysql-server'  # Docker Hub repository name

    # Optionally check the tag we got back
    - name: Check outputs
      run: |
        echo "Pull Request Number - ${{ steps.latest_tag.outputs.tag }}"
```
