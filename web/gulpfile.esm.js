"use strict"

import fs from "fs"
import glob from "glob"
import path from "path"
import zlib from "zlib"

import gulp from "gulp"
import gulpHash from "gulp-hash-filename"
import gulpEsbuild from "gulp-esbuild"
import gulpFavicons from "@flexis/favicons/lib/stream"
import gulpPostcss from "gulp-postcss"
import gulpRename from "gulp-rename"
import gulpSass from "gulp-sass"
import gulpSourcemaps from "gulp-sourcemaps"
import gulpSvgStore from "gulp-svgstore"
import vinylSourceStream from "vinyl-source-stream"

import del from "del"
import mergeStream from "merge-stream"
import sass from "sass"
import through from "through2"

import {stimulusPlugin} from "esbuild-plugin-stimulus"


const DEST=path.resolve("../assets/www")

const sassCompiler = gulpSass(sass)

// hashName returns a gulp stream for hashing filenames with the
// same pattern.
function hashName() {
  return gulpHash({
    format: "{name}.{hash:8}{ext}",
  })
}

// destCompress returns a gulp stream that compresses the current
// stream's files in gzip or brotli.
// It pushes the resulting file to the stream with an added suffix
// (.gz or .br).
function destCompress(format) {
  return through.obj(function(file, _, done) {
    if (file.isNull() || file._isCompressed || path.extname(file.basename) == ".map") {
      return done(null, file)
    }

    if (file.isStream()) {
      done(null, file)
      return
    }

    let options, fn
    let cf = file.clone({deep:true, contents: true})
    cf.basename = `${cf.basename}.${format}`

    if (format == "gz") {
      options = {
        level:9,
      }
      fn = zlib.gzip
    } else if (format == "br") {
      options = {
        params: {
          [zlib.constants.BROTLI_PARAM_QUALITY]: 11,
        },
      }
      fn = zlib.brotliCompress
    } else {
      done(`unknown format ${format}`)
      return
    }

    fn(cf.contents, options, (err, contents) => {
      if (err) {
        done(err)
        return
      }

      cf.contents = contents
      cf._isCompressed = true
      this.push(cf)
      done(null, file)
    })
  })
}

// cleanFiles calls del() with some default options.
function cleanFiles(...args) {
  return del(args, {cwd:DEST, force:true})
}

// clean_js remove the JS assets.
function clean_js(done) {
  return cleanFiles("*.js", "*.js.*")
}

// clean_css removes the CSS assets (and fonts).
function clean_css() {
  return cleanFiles("*.css", "*.css.*", "fonts")
}

// clean_media removes static assets like svg or images.
function clean_media() {
  return cleanFiles("img")
}

// clean_manifest creates an empty manifest.json file.
function clean_manifest() {
  let s = vinylSourceStream("manifest.json")
  s.end("{}")
  return s.pipe(gulp.dest(DEST))
}

// clean delete files in the destination folder
function clean_all(done) {
  return gulp.series(
    clean_js,
    clean_css,
    clean_media,
    clean_manifest,
  )(done)
}

// js_bundle creates the JS bundle file using esbuild.
function js_bundle() {
  return gulp
    .src("src/main.js")
    .pipe(gulpEsbuild({
      sourcemap: "inline",
      outfile: "bundle.js",
      bundle: true,
      format: "esm",
      platform: "browser",
      metafile: false,
      minifyIdentifiers: true,
      minifyWhitespace: true,
      logLevel: "warning",
      plugins: [
        stimulusPlugin(),
      ],
    }))
    .pipe(gulpSourcemaps.init({loadMaps:true})) // This extracts the inline sourcemap
    .pipe(hashName())
    .pipe(gulpSourcemaps.write("."))
    .pipe(destCompress("gz"))
    .pipe(destCompress("br"))
    .pipe(gulp.dest(DEST))
}

// css_bundle creates the CSS bundle.
function css_bundle() {
  const processors = [
    require("postcss-import"),
    require("./ui/plugins/prose"),
    require("tailwindcss"),
    require("postcss-copy")({
      basePath: [
        "ui",
        "node_modules/@fontsource/lora/files",
        "node_modules/@fontsource/public-sans/files",
      ],
      dest: DEST,
      template: (m) => {
        let folder = "."
        if (["woff", "woff2"].includes(m.ext)) {
          folder = "fonts"
        }

        return `${folder}/${m.name}.${m.hash.substr(0, 8)}.${m.ext}`
      },
    }),
    require("autoprefixer"),
    require("cssnano"),
  ]

  return gulp
    .src([
      "ui/index.sass",
    ])
    .pipe(gulpSourcemaps.init())
    .pipe(sassCompiler().on("error", sassCompiler.logError))
    .pipe(gulpRename("bundle.css"))
    .pipe(gulpPostcss(processors))
    .pipe(hashName())
    .pipe(gulpSourcemaps.write("."))
    .pipe(destCompress("gz"))
    .pipe(destCompress("br"))
    .pipe(gulp.dest(DEST))
}

// css_epub create the css file used for epub export
function css_epub() {
  const processors = [
    require("postcss-import"),
    require("./ui/plugins/prose"),
    require("autoprefixer"),
  ]

  return gulp
    .src([
      "ui/epub/stylesheet.sass",
    ])
    .pipe(gulpSourcemaps.init())
    .pipe(sassCompiler().on("error", sassCompiler.logError))
    .pipe(gulpRename("epub.css"))
    .pipe(gulpPostcss(processors))
    .pipe(gulp.dest(DEST))
}

// icons creates the icon sprite file
function icon_sprite() {
  // Icons are defined in this file
  const icons = JSON.parse(fs.readFileSync("./media/icons.json"))

  return gulp
    .src(Object.values(icons))
    .pipe(gulpRename((file, f) => {
      // Set new filename on each entry in order to set
      // a chosen ID on each symbol.
      let p = path.relative(f.cwd, f.path)
      let id = Object.entries(icons).find(x => x[1] == p)[0]
      file.basename = id
    }))
    .pipe(gulpSvgStore())
    .pipe(gulpRename("img/icons.svg"))
    .pipe(hashName())
    .pipe(destCompress("gz"))
    .pipe(destCompress("br"))
    .pipe(gulp.dest(DEST))
}

function generate_favicons() {
  return gulp
    .src("./media/img/favicon.svg")
    .pipe(gulpFavicons({
      icons: {
        android: false,
        apple: true,
        appleStartup: false,
        favicon: true,
      },
      verbose: true,
      headers: false,
    }))
    .pipe(gulp.dest("./media/favicons"))
}

async function generate_rnd_icons(done) {
  let res = []

  let root = path.resolve("./media/rndicons")
  let files = await fs.promises.readdir(root)
  for (let x of files) {
    x = path.join(root, x)
    let c = await fs.promises.readFile(x)
    res.push(c.toString())
  }

  let dest = path.join(DEST, "rndicons.json")
  await fs.promises.writeFile(dest, JSON.stringify(res))

  done()
}

// copy_files copies some files to the destination.
function copy_files() {
  return mergeStream(
    gulp
      .src("media/favicons/*")
      .pipe(hashName())
      .pipe(gulp.dest(path.join(DEST, "img/fi"))),
  )
}

// write_manifest generates a manifest.json file in the destination folder.
// It's a very naive process that lists all the files in the destination
// folder and creates a mapping for all the files having a hash suffix.
async function write_manifest(done) {
  const rxFilename = new RegExp(/^(.+)(\.[a-f0-9]{8}\.)(.+)$/)
  const excluded = [".br", ".gz", ".map"]

  glob(path.join(DEST, "**/*"), {}, async (err, files) => {
    if (err) {
      done(err)
      return
    }
    let manifest = {}
    for (let f of files) {
      let st = await fs.promises.stat(f)
      if (!st.isFile()) {
        continue
      }
      f = path.relative(DEST, f)
      if (f == "manifest.json" || excluded.includes(path.extname(f))) {
        continue
      }

      let m = f.match(rxFilename)
      if (!m) {
        continue
      }

      manifest[`${m[1]}.${m[3]}`] = f
    }

    let dest = path.join(DEST, "manifest.json")
    fs.writeFile(dest, JSON.stringify(manifest, null, "  ") + "\n", done)
  })
}


// ------------------------------------------------------------------
// Gulp pipelines
// ------------------------------------------------------------------

const full_build = gulp.series(
  clean_all,
  gulp.parallel(
    js_bundle,
    icon_sprite,
    generate_rnd_icons,
    copy_files,
    css_bundle,
    css_epub,
  ),
  write_manifest,
)

function watch_js() {
  gulp.watch(
    ["src/**/*"],
    gulp.series(
      clean_js,
      js_bundle,
      write_manifest,
    ),
  )
}

function watch_css() {
  gulp.watch(
    ["tailwind.config.js", "ui/**/*", "../assets/templates/**/*.jet.html"],
    gulp.series(
      clean_css,
      css_bundle,
      css_epub,
      write_manifest,
    ),
  )
}

function watch_media() {
  gulp.watch(
    ["media/**/*"],
    gulp.series(
      clean_media,
      icon_sprite,
      copy_files,
      write_manifest,
    ),
  )
}

exports.clean = clean_all
exports.js = js_bundle
exports.css = gulp.series(clean_css, css_bundle, css_epub)
exports.epub = css_epub
exports.icons = icon_sprite
exports.rndicons = generate_rnd_icons
exports.favicons = generate_favicons
exports.copy = copy_files

exports["watch:css"] = watch_css
exports["watch:js"] = watch_js

exports.dev = gulp.series(
  full_build,
  gulp.parallel(
    watch_js,
    watch_css,
    watch_media,
  ),
)

exports.default = full_build
