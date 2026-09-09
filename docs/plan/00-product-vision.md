# InfoBoard: Product Vision

## One-line definition

**InfoBoard is a local-first text intelligence dashboard for capturing, organizing, and revisiting personal information.**

It is not primarily a document reader or a generic note-taking application. Its core job is to turn scattered text into a dashboard that makes information visible, searchable, and connected.

## The problem

Useful information is usually scattered across web articles, PDFs, Markdown files, copied text, and short personal notes. Traditional bookmark tools preserve links but make recall difficult; ordinary folders preserve files but do not reveal relationships between their contents.

InfoBoard keeps a local snapshot of the content, places it in user-defined collections/projects, and makes it possible to find items by exact words or by meaning.

## Target user and principles

The initial product serves one user on their own machine.

- **Local-first:** no account, sync, or shared workspace is required.
- **Text-first:** the primary content is extracted text; web images are out of scope for the MVP.
- **Organized around projects:** an item can be connected to multiple collections.
- **Useful dashboard, not vanity metrics:** analytics should help the user see what they saved, read, and are accumulating.
- **Private by default:** local embeddings are the default; a cloud embedding provider is opt-in.

## Core information object

An **item** represents a saved piece of information. It can originate from a public web article, PDF, Markdown file, plain-text file, or manually added text. Every item has a title, content snapshot, source metadata, reading state, collections, an optional personal note, and an indexing state.

## MVP outcome

A user can save an article or file, return later to its local text snapshot, organize it into multiple projects, leave a note, search it by wording or meaning, and inspect a dashboard of their library.

