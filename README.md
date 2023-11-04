# mcgen - Minecraft Achievement Generator
mcgen is a Golang-based JSON web API for generating custom Minecraft achievements.


## Usage
A demo of the API is available at https://mcgen.menzerath.eu.

### Requests

#### GET `/api/v1/achievement`
```
/api/v1/achievement?background=sword_diamond&title=Achievement%20Title&text=Achievement%20Text
```

#### POST `/api/v1/achievement`
```json
{
    "background": "sword_diamond",
    "title": "Achievement Title",
    "text": "Achievement Text"
}
```

#### GET `a.php`
We also support the legacy api of https://github.com/menzerath/minecraft-achievement-generator.
```
/a.php?i=3&h=Achievement%20Title&t=Achievement%20Text
```

#### GET `/a/:background/:title/:text`
We also support the legacy api of https://github.com/menzerath/minecraft-achievement-generator.
```
/a/3/Achievement%20Title/Achievement%20Text
```

### Icons
Available icons are listed in [this](assets/backgrounds) directory.  
Use their filename without the `.png` extension as the `background` parameter.

### Download
To download an image, set the `output` parameter to `download`.  
This is either in the query string or the JSON body.


## Installation
Grab a current release for your platform and run the executable.  
An http server will be exposed on port 8080 and serve the API.

### Docker
Grab a current docker image from the [GitHub Container Registry](https://github.com/menzerath/mcgen/pkgs/container/mcgen).  
An http server will be exposed on port 8080 and serve the API.

### Monitoring
Prometheus metrics are available at `localhost:9100/metrics`.


## License
"Minecraft" is a trademark of Notch Development AB.
