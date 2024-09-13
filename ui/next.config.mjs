/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "standalone",
  eslint: {
    ignoreDuringBuilds: true,
  },
  transpilePackages: ["lucide-react"],
};

export default nextConfig;
