import type { Metadata } from "next";
import Navbar from "@/components/layout/Navbar";
import PropertyCard from "@/components/common/PropertyCard";
import "./globals.css";

export const metadata: Metadata = {
  title: "GoFullStock | Find Rental Houses Near You",
  description:
    "Browse verified rental houses including 1 bedroom, 2 bedroom, and studio apartments near your location. Find vacant houses fast with GoFullStock.",
  keywords: [
    "rental houses",
    "houses near me",
    "1 bedroom apartment",
    "2 bedroom apartment",
    "house rent",
    "Kenya-kibabii rentals",
  ],
  authors: [{ name: "GoFullStock Team" }],
  openGraph: {
    title: "GoFullStock | Smart House Rentals",
    description:
      "Discover available rental houses near you. Search by location, category, and availability status.",
    url: "https://gofullstock.com",
    siteName: "GoFullStock",
    images: [
      {
        url: "./images/4.jpeg",
        width: 1200,
        height: 630,
        alt: "GoFullStock rental houses",
      },
    ],
    type: "website",
  },
  robots: {
    index: true,
    follow: true,
  },
};

const mockProperties = [
  {
    id: 1,
    title: "Modern 2 Bedroom Apartment",
    location: "butieli",
    type: "2 Bedroom",
    status: "vacant",
    image: "./images/1.jpeg",
    
  },
  {
    id: 2,
    title: "Affordable 1 Bedroom",
    location: "namaloko",
    type: "1 Bedroom",
    status: "occupied",
    image: "./images/2.jpeg",
  },
  {
    id: 3,
    title: "Spacious Studio",
    location: "kibabii village",
    type: "Studio",
    status: "vacant",
    image: "./images/3.jpeg",
  },
];

export default function HomePage() {
  return (
    <>
      <Navbar />

      <section className="hero">
        <div className="hero-overlay">
          <h1>Find a Home Near You</h1>
          <p>Browse verified rental houses by location and category</p>

          <div className="search-box">
            <input type="text" placeholder="Search by location..." />
            <select>
              <option value="">House Category</option>
              <option value="1 Bedroom">1 Bedroom</option>
              <option value="2 Bedroom">2 Bedroom</option>
              <option value="Studio">Studio</option>
            </select>
            <button>Search</button>
          </div>
        </div>
      </section>

      <section className="properties-section">
        <h2>Available Houses</h2>

        <div className="properties-grid">
          {mockProperties.map((property) => (
            <PropertyCard key={property.id} property={property} />
          ))}
        </div>
      </section>
    </>
  );
}