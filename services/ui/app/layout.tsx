
import "./globals.css";
export const metadata = { title: "OwlerLite v2", description: "Scopes + Freshness + CTR + KG" };
export default function RootLayout({children}:{children:React.ReactNode}){
  return <html lang="en"><body className="min-h-screen bg-neutral-50">{children}</body></html>
}
