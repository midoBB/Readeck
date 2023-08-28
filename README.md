# Readeck

Readeck is a simple web application that lets you save the
precious readable content of web pages you like and want to keep
forever. \
See it as a bookmark manager and a read later tool.

![Readeck Bookmark List](./screenshots/bookmark-list.png)

## Features

### 🔖 Bookmarks

Like a page you're reading? Paste the link in readeck and you're done!


### 📸 Articles, pictures and videos

Readeck saves the readable content of web pages for you to read later. It also detects when a page is an image or a video and adapts its process accordingly.


### ⭐ Labels, favorites, archives

Move bookmarks to archives or favorites and add as many labels as you want.


### 🖍️ Highlights

Highlight the important content of your bookmarks to easily find it later.


### 🗃️ Collections

If you need a dedicated section with all your bookmarks from the past 2 weeks labeled with "cat", Readeck lets you save this search query into a collection so you can access it later.


### 📖 Ebook export

What's best than reading your collected articles on your e-reader? You can export any article to an ebook file (EPUB). You can even export a collection to a single book!

On top of that, you can directly access Readeck's catalog from your e-reader if it supports OPDS.


### 🔎 Full text search

Whether you need to find a vague piece of text from an article, or all the articles with a specific label or from a specific website, we've got you covered!


### 🚀 Fast!

Readeck is a modern take on so called boring, but proven, technology pieces. It garanties very quick response time and a smooth user experience.


### 🔒 Built for your privacy and long term archival

Will this article you like be online next year? In 10 year? Maybe not; maybe it's all gone, text and images. For this reason, and for your privacy, text and images are all stored in your readeck instance the moment you save a link.

With the exception of videos, not a single request is made from your browser to an external website.

## How to install

Done reading my marketing stuff? Good! Want to try Readeck on your laptop or a server? Even better!

- Go to the [packages](https://codeberg.org/readeck/readeck/packages) page and grab the binary release matching your system,
- Rename the file to `readeck` (or anything you fancy),
- Move this file to the directory you just created,
- Go to the directory and launch the `readeck serve` command.


```bash
cd readeck
./readeck serve
```

The first time you launch Readeck, you'll have to create a user (you!) and then be on your way.

At the end of this short process, Readeck start and is accessible on:

**[http://127.0.0.1:5000/](http://127.0.0.1:5000/)**


## Under the hood

Readeck was born out of frustration (and covid lockdown) from the tools that don't let you have a full archive of the content you save. This key principle guided every step of Readeck development.

### The ZIP file

Every bookmark is stored in a single, immutable, ZIP file. Parts of this file (HTML content, images, etc.) are served directly by the application or converted to a web page or an ebook when needed.

### A simple database

Readeck has a very simple database schema with a few tables and uses a lot of JSON fields when appropriate. The recommended database engine is SQLite for an installation of a few users. It's very likely you'll hit other botllenecks before you encounter database related performance.
