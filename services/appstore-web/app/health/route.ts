/**
 * The liveness probe the compose healthcheck and nginx use.
 *
 * It sits outside the language segments on purpose — a load balancer asking
 * whether this process is up should not be redirected into Mongolian first —
 * which is why the proxy's matcher excludes it.
 */
export function GET() {
  return Response.json({ status: "ok", service: "appstore-web" });
}
