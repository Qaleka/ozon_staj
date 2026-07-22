# Posts & Comments — GraphQL-сервис

## Запуск

### Docker (Postgres)

```bash
docker compose up --build
```

### Docker (in-memory)

```bash
STORAGE=memory docker compose up --build app
```

### Локально

```bash
go run ./cmd/main
```

По умолчанию используется in-memory хранилище; параметры — через переменные
окружения или `.env` (см. `.env.example`):

После запуска: GraphQL Playground — http://localhost:8080, эндпоинт — `POST /query`,
подписки — WebSocket на том же `/query`.

## Тесты

```bash
go test ./...
```

Покрытие:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out    # разбивка по функциям
go tool cover -html=coverage.out    # HTML-отчёт в браузере
```

## Примеры запросов

Создать пост:

```graphql
mutation {
  createPost(input: {title: "Первый пост", content: "Привет!", author: "alice"}) {
    id title author commentsDisabled createdAt
  }
}
```

Оставить комментарий:

```graphql
mutation {
  createComment(input: {postId: "<POST_ID>", author: "bob", text: "Отличный пост"}) {
    id text createdAt
  }
}
```

Запретить комментарии (только автор поста):

```graphql
mutation {
  setCommentsDisabled(postId: "<POST_ID>", author: "alice", disabled: true) {
    id commentsDisabled
  }
}
```

Список постов и дерево комментариев с пагинацией на каждом уровне:

```graphql
query {
  posts(first: 10) {
    edges { node { id title } cursor }
    pageInfo { endCursor hasNextPage }
  }
}

query {
  post(id: "<POST_ID>") {
    title
    comments(first: 10) {
      edges {
        node {
          id text author
          replies(first: 5) {
            edges { node { id text } }
            pageInfo { endCursor hasNextPage }
          }
        }
        cursor
      }
      pageInfo { endCursor hasNextPage }
    }
  }
}
```

Подписка на новые комментарии поста:

```graphql
subscription {
  commentAdded(postId: "<POST_ID>") {
    id text author parentId createdAt
  }
}
```