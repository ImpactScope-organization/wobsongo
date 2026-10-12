# WobSongo Dashboard Technical Specification

This is the technical specification for the WobSongo dashboard application, which outlines the architecture, technologies, and implementation details required to build and maintain the application.

The stack is SvelteKit, openapi, tailwindcss + shadcn-svelte, and node adapter.

WobSongo is a fact-checking system powered by AI in the domain of Sexual and Reproductive Health (SRH). The dashboard is one of the main interface for power users to manage and monitor the system. The main data is sourced from social media and WhatsApp bot interactions by users.

## Development guidelines

### Components

We use standard Svelte components with TypeScript. Components should be reusable and isolated, with clear and intuitive APIs. Components should be designed to be responsive and accessible, following best practices for web development. Components do not perform business logic nor data fetching. They only receive props and emit events. Components should be tested using unit tests and integration tests to ensure they work as expected. Every components should have Storybook stories for documentation and testing purposes.

### Styling

We use TailwindCSS for styling. We follow a consistent design system and use utility classes to style components. We avoid using inline styles and prefer to use CSS classes for styling. We also use CSS variables for theming and customization. The main CSS file is layout.css

### Component library

We use shadcn-svelte as component library. This provides a set of pre-built components that can be used to quickly build the application's user interface. We customize the components as needed to fit the application's design system and branding. We also contribute back to the component library by submitting bug fixes and feature requests. This is a copy-paste component library, not importable. We will copy the components we need into our project and customize them as needed, installed via CLI. The docs is available here https://shadcn-svelte.com/llms.txt

There are also pre-build components from shadcn-svelte called blocks, like login and signup forms/page, all ready to use.

Charts are also available from shadcn-svelte, no need for other libraries. It is not installable via CLI, but we can copy-paste the components we need into our project and customize them as needed.

### Server-rendering

We use server-side rendering almost all the time, unless there's a specific reason to use client-side rendering. We use SvelteKit's built-in server-side rendering capabilities to render pages on the server and send them to the client. This improves performance and SEO, as well as provides a better user experience. All data fetching should be done on the server-side, and the client should only receive the rendered HTML and any necessary data. Every static pages should be prerendered. We use SvelteKit's built-in prerendering capabilities to generate static HTML pages for the application. This improves performance and reduces server load, as well as provides a better user experience. All static pages should be generated at build time, and any dynamic content should be fetched on the server-side.

### System architecture

We split the project into three main layers: the presentation layer, the application layer, and the data layer. The presentation layer is responsible for rendering the user interface and handling user interactions. The application layer is responsible for implementing the business logic and coordinating between the presentation and data layers. We call this the service layer. The data layer is responsible for managing the application's data and providing an interface for the application layer to access it.

#### The presentation layer

This is basically a collection of Svelte components, pages, and page server files working together to fetch and render pages.

#### The service layer

This is the main business logic layer of the application. It is responsible for implementing the application's use cases and coordinating between the presentation and data layers. The service layer should be designed to be modular and extensible, allowing for easy addition of new features and functionality. The service layer should also be designed to be testable, with clear and intuitive APIs that can be easily mocked and tested in isolation. The dependency on the data layer should be abstracted as an interface, allowing for easy swapping of data sources and implementations. The service layer should also handle any necessary data validation and error handling, ensuring that the application behaves correctly and gracefully in the face of unexpected input or errors.

Since this is a SaaS, all methods should be scoped per tenant, while considering sharing feature. For example, if Alex is the main tenant user and the account is shared with Beth, then all Beth's requests would have Alex's tenant ID as the main tenant. To achieve this, we will likely need to track tenant sharing in a separate table, and use this information to determine the appropriate tenant ID for each request. This will ensure that all data is properly scoped and isolated per tenant, while still allowing for sharing of data between legitimate users.

#### The domain modeling layer

This package contains the domain models and business logic of the application. It defines the entities, value objects, and aggregates that make up the application's domain, as well as the rules and constraints that govern their behavior. The domain modeling layer should be designed to be independent of any specific technology or framework, allowing for easy testing and maintenance. The domain modeling layer should also be designed to be extensible, allowing for easy addition of new entities and business rules as the application evolves.

#### The data layer (or the repo layer)

This layer is responsible for managing the application's data and providing an implementation of the repository interfaces defined in the service layer. In the initial phase of development, we will use static stub data layer that mimics the behavior of a real data source without persistence. For example, if the method is `GetById(id: string): Promise<User>`, the stub will define conventions around the behavior based on the given ID. For instance, if the ID ends with "000", we give the user. If the ID ends with "404" we return a not found error. If the ID ends with "500", we return an internal server error. This allows us to test the service layer without relying on a real data source, and to simulate different scenarios and edge cases.

The real repo layer will openapi client generated from the backend swagger.json file at `..`, powered by Go. We'll work on this later towards the end of the project.

#### Utils and helpers

This contains utility functions and helper classes that are used throughout the application. These should be designed to be reusable and modular, with clear and intuitive APIs that can be easily tested and maintained. Utility functions should be designed to be pure and side-effect-free, allowing for easy testing and debugging. Helper classes should be designed to encapsulate common functionality and provide a clear interface for interacting with the application's data and services.

One important helper is to add an independence from the wall clock, so we can test time-dependent features with ease. Abstract time provider behind an interface, and use dependency injection to provide the implementation. This will allow us to easily swap out the implementation for testing purposes, without affecting the rest of the application.

#### QA

Ensure all linters, formatters, and type checkers are in place and configured correctly. We will use ESLint for linting, Prettier for formatting, and TypeScript for type checking. These will need to run after every implementation and before committing to git.

#### Git

Agents will not handle git commits. We will use conventional commits for commit messages, following the format of `<type>(<scope>): <description>`. The types can be `feat`, `fix`, `docs`, `style`, `refactor`, `test`, or `chore`. The scope should be the name of the component or module being modified. The description should be a brief summary of the changes made. Only the user handles git operations, including commits, pushes, and merges. Agents will only have a read-only access to git via Gitlab's `glab` CLI tool and `git` itself.

## Authentication and authorization

We will support 2 methods: regular email/password and Google sign-in. We will use JWT tokens for authentication and authorization, with access tokens and refresh tokens. Access tokens will have a short expiration time (e.g., 15 minutes), while refresh tokens will have a longer expiration time (e.g., 7 days). We will store the access token in memory and the refresh token in an HTTP-only cookie to prevent XSS attacks. We will also implement role-based access control (RBAC) to restrict access to certain features and resources based on the user's role.

This is a later feature, for now we prioritize email/password login. We will implement Google sign-in later, after the initial release.

## Progressive Web App (PWA)

The application will be built as a Progressive Web App (PWA), allowing users to install the app on their devices and access it offline. We will use service workers to cache static assets and API responses, allowing the app to function even when the user is offline. We will also implement push notifications to keep users informed about important events and updates.

## Security

Ensure that the app is secure and follows best practices for web application security. This includes implementing HTTPS, input validation, and protection against common web vulnerabilities such as cross-site scripting (XSS) and SQL injection. We will also implement rate limiting and logging to prevent abuse and monitor for suspicious activity.

AI-related vulnerabilities should be considered, such as prompt injection attacks. We will implement input validation and sanitization to prevent malicious input from being processed by the AI model. We will also implement logging and monitoring to detect and respond to any suspicious activity related to the AI model.
