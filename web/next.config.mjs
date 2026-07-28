/** @type {import("next").NextConfig} */
const nextConfig = {
  async headers() {
    return [
      {
        source: "/:path*",
        headers: [
          {
            key: "Content-Security-Policy",
            value:
              "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https://lh3.googleusercontent.com; connect-src 'self' " +
              (process.env.NEXT_PUBLIC_API_BASE_URL ?? "/api") +
              "; frame-ancestors 'none'; base-uri 'self'; form-action 'self'; object-src 'none'",
          },
          {
            key: "X-Frame-Options",
            value: "DENY",
          },
          {
            key: "X-Content-Type-Options",
            value: "nosniff",
          },
          {
            key: "Referrer-Policy",
            value: "strict-origin-when-cross-origin",
          },
          {
            key: "Permissions-Policy",
            value:
              "camera=(), microphone=(), geolocation=(), publickey-credentials-create=(self), publickey-credentials-get=(self)",
          },
        ],
      },
    ];
  },
  images: {
    remotePatterns: [
      {
        protocol: "https",
        hostname: "lh3.googleusercontent.com",
      },
    ],
  },
  reactCompiler: true,
};

export default nextConfig;
