import { useState } from "react";
import { LoginForm } from "@/components/auth/LoginForm";
import { SignupForm } from "@/components/auth/SignupForm";
import { Button } from "@/components/ui/button";

const Index = () => {
  const [isLogin, setIsLogin] = useState(true);

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-gradient-to-br from-match-light/10 to-match-dark/10 p-4">
      <div className="w-full max-w-md bg-white rounded-xl shadow-lg p-8 animate-fade-in">
        <div className="text-center mb-8">
          <h1 className="text-4xl font-bold mb-8">Grant Matcherator</h1>
          <p className="text-gray-600 mt-2">
            {isLogin ? "Welcome back!" : "Join our community"}
          </p>
        </div>

        {isLogin ? <LoginForm /> : <SignupForm />}

        <div className="mt-6 text-center">
          <Button
            variant="link"
            onClick={() => setIsLogin(!isLogin)}
            className="text-gray-600 hover:text-match-light"
          >
            {isLogin ? "Need an account? Sign up" : "Already have an account? Login"}
          </Button>
        </div>
      </div>
    </div>
  );
};

export default Index;