import { useEffect, useState } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { MessageSquare, UserRound, Handshake, BellDot, Users } from "lucide-react";
import { useToast } from "@/hooks/use-toast";
import { useMutation } from "@tanstack/react-query";
import { apiRequest } from "@/lib/api";
import { cn } from "@/lib/utils";

interface Message {
  id: string;
  sender_id: number;
  content: string;
  timestamp: string;
  read: boolean;
}

export const Header = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const [ws, setWs] = useState<WebSocket[]>([]);
  const [unreadMessages, setUnreadMessages] = useState(0);
  const [newMatches, setNewMatches] = useState(0);
  const [isConnecting, setIsConnecting] = useState(true);
  const [localMessages, setLocalMessages] = useState<Message[]>([]);

  // Fetch initial notifications
  useQuery({
    queryKey: ["notifications"],
    queryFn: async () => {
      console.log("Fetching initial notifications...");
      const res = await apiRequest("/notifications");
      console.log("Initial notifications fetched:", res);
      setUnreadMessages(res.unreadMessages);
      setNewMatches(res.newMatches);
      return res;
    },
    enabled: !!localStorage.getItem("token"), // Only run if token exists
  });

  // Mark notifications as read
  const markNotificationsAsRead = useMutation({
    mutationFn: () => apiRequest("/notifications/mark-read", { method: "POST" }),
    onSuccess: () => {
      console.log("Notifications marked as read successfully.");
      setUnreadMessages(0);
    },
  });

  // Establish WebSocket connection
  useEffect(() => {
    let reconnectTimeout: NodeJS.Timeout;
    const newWs: WebSocket[] = [];

    const connectWebSocket = async () => {
      try {
        const token = localStorage.getItem("token");
        if (!token) {
          console.log("No token found, skipping WebSocket connections");
          return;
        }

        // Get existing connections to find match IDs
        try {
          const connectionsResponse = await apiRequest("/connections");
          console.log("Connections response:", connectionsResponse);
          
          if (Array.isArray(connectionsResponse)) {
            // Close any existing WebSocket connections
            newWs.forEach(ws => {
              if (ws.readyState === WebSocket.OPEN) {
                ws.close();
              }
            });
            newWs.length = 0;

            // Get chat preferences first
            try {
              const chatPrefs = await apiRequest("/chat/preferences");
              console.log("Chat preferences:", chatPrefs);
              
              if (!chatPrefs.opt_in) {
                console.log("Chat is not enabled for this user");
                return;
              }

              // Create new connections
              connectionsResponse.forEach((connection) => {
                console.log(`Processing connection:`, connection);
                if (connection.id) {
                  console.log(`Connecting to chat WebSocket for match ${connection.id}`);
                  const wsUrl = `${import.meta.env.VITE_WS_URL}/ws/chat/${connection.id}?token=Bearer ${token}`;
                  const chatWs = new WebSocket(wsUrl);
                  
                  chatWs.onopen = () => {
                    console.log(`Chat WebSocket connected for match ${connection.id}`);
                  };

                  chatWs.onerror = (error) => {
                    console.error(`Chat WebSocket error for match ${connection.id}:`, error);
                    // Don't retry on error, just log it
                  };

                  chatWs.onclose = () => {
                    console.log(`Chat WebSocket closed for match ${connection.id}`);
                    // Remove from array
                    const index = newWs.indexOf(chatWs);
                    if (index > -1) {
                      newWs.splice(index, 1);
                    }
                  };

                  newWs.push(chatWs);
                }
              });
            } catch (error) {
              console.error("Error fetching chat preferences:", error);
            }
          }
        } catch (error) {
          console.error("Error fetching connections:", error);
        }

        // Connect to notifications WebSocket
        const notificationsWsUrl = `${import.meta.env.VITE_WS_URL}/ws/notifications?token=Bearer ${token}`;
        const notificationsWs = new WebSocket(notificationsWsUrl);
        
        notificationsWs.onopen = () => {
          console.log("Notifications WebSocket connected");
        };

        notificationsWs.onerror = (error) => {
          console.error("Notifications WebSocket error:", error);
          // Don't retry on error, just log it
        };

        notificationsWs.onclose = () => {
          console.log("Notifications WebSocket closed");
          // Remove from array
          const index = newWs.indexOf(notificationsWs);
          if (index > -1) {
            newWs.splice(index, 1);
          }
        };

        newWs.push(notificationsWs);

      } catch (error) {
        console.error("Error in WebSocket connection setup:", error);
      }
    };

    connectWebSocket();

    // Cleanup function
    return () => {
      if (reconnectTimeout) {
        clearTimeout(reconnectTimeout);
      }
      // Close all WebSocket connections
      newWs.forEach(ws => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.close();
        }
      });
    };
  }, []); // Empty dependency array since we want this to run once on mount

  const handleLogout = () => {
    console.log("Logging out...");
    localStorage.removeItem("token");
    localStorage.removeItem("user");
    toast({ title: "Logged out successfully", description: "See you next time!" });
    navigate("/");
  };

  const handleNavigateToChats = async () => {
    console.log("Attempting to navigate to /chats...");

    // Wait until the WebSocket connections are established before navigating
    if (isConnecting) {
      console.log("WebSocket is still connecting, cannot navigate to chats yet.");
      return;
    }

    console.log("WebSocket connection established, proceeding to navigate to /chats...");
    console.log("Marking notifications as read...");
    await markNotificationsAsRead.mutateAsync();
    navigate("/chats");
  };

  return (
    <nav className="bg-white shadow-md p-4">
      <div className="max-w-4xl mx-auto flex justify-between items-center">
        <h1
          onClick={() => {
            console.log("Navigating to dashboard...");
            navigate("/dashboard");
          }}
          className="text-2xl font-bold bg-gradient-to-r from-match-light to-match-dark text-transparent bg-clip-text cursor-pointer hover:opacity-90 transition-opacity"
        >
          Grant Matcherator
        </h1>
        <div className="flex gap-4">
          <Button 
            variant="ghost" 
            className={cn(
              "flex items-center gap-2",
              location.pathname === "/dashboard" && "bg-match-light/10 text-match-dark"
            )} 
            onClick={() => {
              console.log("Navigating to dashboard...");
              navigate("/dashboard");
            }}
          >
            <Users className="w-4 h-4" />
            Matches
          </Button>
          <Button 
            variant="ghost" 
            className={cn(
              "flex items-center gap-2",
              location.pathname === "/profile" && "bg-match-light/10 text-match-dark"
            )} 
            onClick={() => {
              console.log("Navigating to profile...");
              navigate("/profile");
            }}
          >
            <UserRound className="w-4 h-4" />
            Profile
          </Button>
          <Button 
            variant="ghost" 
            className={cn(
              "flex items-center gap-2 relative",
              location.pathname === "/matches" && "bg-match-light/10 text-match-dark"
            )} 
            onClick={() => {
              console.log("Navigating to matches...");
              navigate("/matches");
            }}
          >
            <Handshake className="w-4 h-4" />
            Connections
            {newMatches > 0 && (
              <span className="absolute -top-1 -right-1 bg-red-500 text-white text-xs rounded-full w-5 h-5 flex items-center justify-center">
                {newMatches}
              </span>
            )}
          </Button>
          <Button
            variant="ghost"
            className={cn(
              "flex items-center gap-2 relative",
              location.pathname === "/chats" && "bg-match-light/10 text-match-dark"
            )}
            onClick={handleNavigateToChats}
          >
            {unreadMessages > 0 ? (
              <BellDot className="w-4 h-4 text-red-500" />
            ) : (
              <MessageSquare className="w-4 h-4" />
            )}
            Chats
            {unreadMessages > 0 && (
              <span className="absolute -top-1 -right-1 bg-red-500 text-white text-xs rounded-full w-5 h-5 flex items-center justify-center">
                {unreadMessages}
              </span>
            )}
          </Button>
          <Button onClick={handleLogout} variant="outline" className="hover:text-match-dark">
            Logout
          </Button>
        </div>
      </div>
    </nav>
  );
};
