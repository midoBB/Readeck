# Readeck Content Scripts

## API

The main content script API consists in exporting some functions that can perform operations on the current extracted information.

### priority

`exports.priority = 0`

This is a integer value, defaulting to `0` when unset. The higher the number, the later the script will run. For a script overriding the site configuration with `setConfig`, you'll need to set it to a value higher than `10` to ensure the script runs last.

### isActive

`exports.isActive()`

This function must return a boolean to indicate that the script can run in the current context. \
If the function is absent from the script, the other functions will never run.

```js
// Always run
exports.isActive = function() {
  return true
}

// Only run on a specific domain
exports.isActive = function() {
  return $.domain == "youtube.com"
}
```

### setConfig

`exports.setConfig(config)`

This function receives an SiteConfiguration object reference. It can set properties of the object as long as the value types don't change.

```js
exports.setConfig = function(config) {
  // Override TitleSelectors
  config.titleSelectors = ["/html/head/title"]

  // Append a body selector
  config.bodySelectors.push("//main")
}
```

### processMeta

`exports.processMeta()`

This function runs after loading the page meta data.

## Global variables and functions

### `$`: extractor information

The global variable `$` holds everything that's needed to read or change information on the current extraction process.

#### `$.domain` (read only)

The domain of the current extraction. Note that it's different from the host name. For example, if the host name is `www.example.co.uk`, the value of `$.domain` is `example.co.uk`.

The value is always in its Unicode form regardless of the initial input.

#### `$.host` (read only)

The host name of the current extraction.

The value is always in its Unicode form regardless of the initial input.

#### `$.url` (read only)

The URL of the current extraction. The value is a string that you can parse with `new URL($.url)` when needed.

#### `$.meta`

This variable is an object whose values are lists of strings. For example:

```js
{
  "html.title": ["document title"]
}
```
You can read, set or delete any value in `$.meta`. You can **not** use `push()` to add new values.

When setting values, you can use a list or a single string.

```js
$.meta["html.title"] = "new title" // valid
$.meta["html.author"] = ["someone", "someone else"] // valid
```

#### `$.authors`

A list of found authors in the document.

**Note**: When setting this value, it must be a list and you can **not** use `$.authors.push()` to add new values.

#### `$.description`

A string with the document description.

#### `$.title`

A string with the document title.

#### `$.type`

The document type. When settings this value, it must be one of "article", "photo" or "video".

#### `$.html` (write only)

When settings a string to this variable, the whole extracted content is replaced. This is an advanced option and should only be used for content that are not articles (photos or videos).

#### `$.readability`

Whether readability is enabled for this content. It can be useful to set it to false when setting an HTML content with `$.html`.

Please note that even though readability can be disabled, it won't disable the last cleaning pass that removes unwanted tags and attributes.

### unescapeURL

```js
/**
 * @param {string} value - input URL
 * @return {string}
 */
function unescapeURL(value)
```

This function transforms an escaped URL to its non escaped version.

### decodeXML

```js
/**
 * @param {string} input
 * @return {Object}
 */
function decodeXML(input)
```

This function decodes an XML text into an object than can be serialized into JSON or filtered.


### requests

If you need to perform HTTP requests in a content script, you must use the `requests` global object.

This is by no means a full featured or advanced HTTP client but it will let you perform simple requests and retrieve JSON or text responses.

```js
const rsp = requests.get("https://nativerest.net/echo/get")
rsp.raiseForStatus()
const data = rsp.json()
```

#### `requests.get(url, [headers])`

This function performs a GET HTTP request and returns a response object.

An optional `header` object can take header values for the request.

#### `requests.post(url, data, [headers])`

This function performs a POST HTTP requests and returns a response object. The `data` parameter **must** be a string of the data you want to send.

An optional `header` object can take header values for the request.

```js
const rsp = requests.post(
  "http://example.net/",
  JSON.stringify({"a": "abc"}),
  {"Content-Type": "application/json"},
)
```

#### `response` object

##### `response.status`

This is the numeric status code.

##### `response.headers`

This contains all the response's headers.

##### `response.raiseForStatus()`

This function will throw an error if the status is not 2xx.

##### `response.json()`

This function returns an object that's the serialization of the response's body.

##### `response.text()`

This function returns the response's text content.


## Types

### Site Configuration

The `setConfig` function receives a `config` object that can be modified.

#### `config.titleSelectors` - []string

XPath selectors for the document title.

#### `config.bodySelectors` - []string

XPath selectors for the document body.

#### `config.dateSelectors` - []string

XPath selectors for the document date.

#### `config.authorSelectors` - []string

XPath selectors for the document authors.

#### `config.stripSelectors` - []string

XPath selectors of elements that must be removed.

#### `config.stripIdOrClass` - []string

List of IDs or classes that belong to elements that must be removed.

#### `config.stripImageSrc` - []string

List of strings that, when present in an `src` attribute of an image will trigger the element removal.

#### `config.singlePageLinkSelectors` - []string

XPath selectors of elements whose `href` attribute refers to a link to the full document.

#### `config.nextPageLinkSelectors` - []string

XPath selectors of elements whose `href` attribute refers to a link to the next page.

#### `config.replaceStrings` - [][2]string

List of pairs of string replacement.

#### `config.httpHeaders` - object

An object that contain HTTP headers being sent to every subsequent requests.
