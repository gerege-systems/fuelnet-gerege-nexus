/*
 * Browser-side error reporting.
 *
 * Off unless NEXT_PUBLIC_SENTRY_DSN is set, which is the default. The DSN is
 * public by design — it identifies the project to write to and grants nothing
 * else — but a build without one ships an SDK that initialises nothing and
 * sends nothing.
 *
 * What is deliberately NOT enabled: Session Replay. It records the DOM of what
 * the person was looking at, and on this platform that is a citizen's
 * registration number, an invoice, or the body of a document waiting for a
 * signature. No amount of masking makes shipping that to an error tracker the
 * right default.
 */

import * as Sentry from '@sentry/nextjs';

const dsn = process.env.NEXT_PUBLIC_SENTRY_DSN;

if (dsn) {
  Sentry.init({
    dsn,
    environment: process.env.NEXT_PUBLIC_ENVIRONMENT ?? 'development',
    release: process.env.NEXT_PUBLIC_RELEASE_VERSION,

    // Errors only. Tracing from the browser would double-report what Tempo
    // already receives from the API, from a source that cannot be trusted about
    // its own timings.
    tracesSampleRate: 0,

    // The SDK's default attaches the request body, the URL and the headers of
    // every fetch that failed. On this platform a URL can carry a single-use
    // verification reference and a body can carry a national identifier.
    sendDefaultPii: false,

    beforeSend(event) {
      if (event.request) {
        // The query string is where single-use references live.
        delete event.request.query_string;
        delete event.request.cookies;
        delete event.request.data;
        if (event.request.headers) {
          delete event.request.headers.Authorization;
          delete event.request.headers.Cookie;
        }
      }
      // Never a person. The tenant is enough to answer "how many organisations
      // does this affect", which is what the grouping is read for.
      if (event.user) {
        event.user = { id: event.user.id };
      }
      return event;
    },

    // Errors the platform did not cause and cannot fix: a browser extension
    // injecting a script, a network that dropped a request mid-flight, the
    // ResizeObserver notice every Chrome build emits. Left in, they are the
    // majority of what an error tracker shows and the reason people stop
    // reading it.
    ignoreErrors: [
      'ResizeObserver loop limit exceeded',
      'ResizeObserver loop completed with undelivered notifications',
      'Non-Error promise rejection captured',
      'NetworkError when attempting to fetch resource',
      'Failed to fetch',
      'AbortError',
    ],
    denyUrls: [/extensions\//i, /^chrome:\/\//i, /^moz-extension:\/\//i],
  });
}

// Next asks for this so it can report navigation timing to the SDK. Exported
// unconditionally: with no DSN it calls into an SDK that does nothing.
export const onRouterTransitionStart = Sentry.captureRouterTransitionStart;
