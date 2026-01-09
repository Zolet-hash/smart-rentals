import Link from 'next/link';
import React from 'react';


const NotFound = () => {
  return (
    <div style={{ textAlign: 'center', marginTop: '50px' }}>
      <h1>404 - Page Not Found</h1>
      <p>Sorry, the page you are looking for does not exist.</p>
      {/* You can add custom styling or a link back to the homepage */}
      <Link href="/">Go to Homepage</Link>
    </div>
  );
};

export default NotFound;
