type PropertyProps = {
  property: {
    title: string;
    location: string;
    type: string;
    status: string;
    image: string;
  };
};

export default function PropertyCard({ property }: PropertyProps) {
  return (
    <div className="property-card">
      <img src={property.image} alt={property.title} />

      <div className="property-info">
        <h3>{property.title}</h3>
        <p className="location">{property.location}</p>
        <p className="type">{property.type}</p>

        <span
          className={
            property.status === "vacant"
              ? "status vacant"
              : "status occupied"
          }
        >
          {property.status}
        </span>
      </div>
    </div>
  );
}
