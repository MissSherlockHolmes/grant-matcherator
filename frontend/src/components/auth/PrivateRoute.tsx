import { Navigate } from "react-router-dom";
import { isAuthenticated } from "@/lib/api";

interface PrivateRouteProps {
  children: React.ReactNode;
  allowedRoles?: ("provider" | "recipient")[];
}

export const PrivateRoute = ({ children, allowedRoles }: PrivateRouteProps) => {
  console.log('=== PrivateRoute Check ===');
  const isAuth = isAuthenticated();
  console.log('Is authenticated:', isAuth);

  if (!isAuth) {
    console.log('Not authenticated, redirecting to home');
    return <Navigate to="/" replace />;
  }

  // If no specific roles are required, allow access
  if (!allowedRoles) {
    console.log('No roles required, allowing access');
    return <>{children}</>;
  }

  // Get user role from localStorage
  const userStr = localStorage.getItem("user");
  console.log('User data from localStorage:', userStr);

  if (!userStr) {
    console.log('No user data found, redirecting to home');
    return <Navigate to="/" replace />;
  }

  const user = JSON.parse(userStr);
  console.log('Parsed user data:', user);

  if (!allowedRoles.includes(user.role)) {
    console.log('User role not allowed, redirecting to dashboard');
    return <Navigate to="/dashboard" replace />;
  }

  console.log('Access granted');
  return <>{children}</>;
};