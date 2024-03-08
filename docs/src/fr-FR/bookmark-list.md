# Bookmarks

Dans les bookmarks se trouve le contenu web que vous appréciez.

## Créer un bookmark

Vous avez trouvé une page que vous aimez ? Parfait, copiez son adresse (lien) dans le champ texte nommé **Enregistrer un lien** sur la [liste des bookmarks](readeck-instance://bookmarks).

![Formulaire de cŕeation de bookmark](./img/bookmark-new.webp)

Après quelques secondes, votre bookmark est prêt. Vous pouvez l'ouvrir pour en consulter le contenu, ajouter des labels, annoter le texte ou exporter un e-book. Pour plus d'information, veuillez lire la section [Détails d'un bookmark](./bookmark).

## Types de bookmark

Readeck reconnaît 3 différents types de contenu web :

### Article

Un article est une page depuis laquelle on peut extraire un contenu text. L'article est affiché comme la version lisible de son contenu.

### Photo

Une photo est une page qui a été reconnue comme un conteneur pour une image (ex. un lien vers Unsplash). Le bookmark affiche l'image enregistée.

### Vidéo

Une vidéo est une page qui a été reconnue comme un conteneur pour une vidéo (ex. un lien vers Youtube ou Vimeo). Le bookmark affiche un lecteur de vidéo. Veuillez noter que les vidéos sont jouées depuis le serveur d'origine.


## Liste des bookmarks

La [liste des bookmarks](readeck-instance://bookmarks) contient tous vos bookmarks.

### Navigation

Dans la barre latérale, se trouvent un champ de recherche et des liens qui vous mènent vers des listes de bookmarks déjà filtrées.

![Bookmark list sidebar](./img/bookmark-sidebar.webp)

- **Recherche** \
  Indiquez ce que vous cherchez (titre, contenu, site web...)
- **Tous** \
  Tous vos bookmarks.
- **Non lus** \
  Les bookmarks qui ne sont pas dans l'archive.
- **Archive** \
  Les bookmarks marqués comme archivés.
- **Favoris** \
  Les bookmarks marqués comme favoris.


Dès que vous commencez à créer des bookmarks, vous verrez les liens suivants :

- **Articles** \
  Les articles
- **Vidéos** \
  Les vidéos
- **Photos** \
  Les photos et images

Enfin, se trouvent 3 autres sections qui mènent vers des pages liées aux bookmarks :

- **[Labels](./labels.md)** \
  Tous les labels ajoutés aux bookmarks
- **Annotations** \
  Toutes les annotations ajoutées aux bookmarks
- **[Collections](./collections.md)** \
  Liste des collections

### Cartes

Chaque élément de la liste des bookmarks est une carte.

![Carte de bookmark](./img/bookmark-card.webp)

Une carte contient:

- le **title** sur lequel vous pouvez cliquer pour accéder au contenu,
- le **nom du site**,
- le **temps de lecture** estimé,
- la **liste des labels**,
- **boutons d'action**

Les buttons permettent d'effectuer les actions suivantes:

- **Favori** \
  Change l'état favori ou non d'un bookmark.
- **Archive** \
  Archive un bookmark (ou le retire des archives).
- **Supprimer** \
  Marque le bookmark pour suppression (cette action peut être annulée pendant quelques secondes).


## Filtrer les bookmarks {#filters}

Sur la liste des bookmarks, vous pouvez filtrer les éléments sur la base de plusieurs critères de recherche. Cliquez sur le bouton "Filtrer la liste" près du titre de la page pour ouvrir le formulaire de recherche.

![Filtres de la liste de bookmarks](./img/bookmark-filters.webp)
Le formulaire de recherche

Entrez un ou plusieurs critères et cliquez sur **Rechercher**.

### Filtres disponibles

Vous pouvez combiner les filtres suivants :

- **Recherche**\
  Recherche dans le texte d'un bookmark, son titre, ses auteurs, le nom du site et du domaine et les labels.
- **Titre**\
  Recherche uniquement dans le titre.
- **Auteur**\
  Recherche uniquement dans la liste des auteurs.
- **Site**\
  Recherche uniquement le nom du site ou son domaine.
- **Label**\
  Recherche des labels donnés.
- **Favori**, **Archivé**, **Type**\
  Ce filtre permet de restreindre la recherche sur ces critères.
- **Depuis**, **Jusqu'à**\
  Ces derniers filtres vous permettent de limiter les résultats par date de création des bookmarks. Par example, vous pouvez obtenir la liste des bookmarks créés pendant les 4 dernières semaines mais pas après la dernière semaine.

### Search query

The **Search**, **Title**, **Author**, **Site** and **Label** fields understand search criteria the same way:

- `startled cat` will find the content with the words **playing** and **cat**
- `"startled cat"` will find the content with the exact words **playing cat** together.
- `cat*` will find the content with the words starting with **cat** (cat, catnip and caterpillar would be a match).
- `-startled cat` will find the content with the word **cat** but NOT the word **startled**.

Après avoir effectué une recherche, vous pouvez l'enregistrer en tant que [collection](./collections.md) pour la rendre permanente.
