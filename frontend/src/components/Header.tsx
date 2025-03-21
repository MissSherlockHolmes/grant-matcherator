import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { MessageSquare, UserRound, Heart, BellDot, Users } from "lucide-react";
import { useToast } from "@/hooks/use-toast";
import { useMutation } from "@tanstack/react-query";
import { apiRequest } from "@/lib/api";

interface Message {
  id: string;
  sender_id: number;
  content: string;
  timestamp: string;
  read: boolean;
}


export const Header = () => {
  const navigate = useNavigate();
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const [ws, setWs] = useState<WebSocket[]>([]); // State for WebSocket connections
  const [unreadMessages, setUnreadMessages] = useState(0);
  const [newMatches, setNewMatches] = useState(0);
  const [isConnecting, setIsConnecting] = useState(true); // Loading state for WebSocket connections
  const [localMessages, setLocalMessages] = useState<Message[]>([]);
  
  // Fetch initial notifications **(Runs only once)** 
  useQuery({
    queryKey: ["notifications"],
    queryFn: async () => {
      //console.log("Fetching initial notifications...");
      const res = await apiRequest("/api/notifications");
      //console.log("Initial notifications fetched:", res);
      setUnreadMessages(res.unreadMessages);
      setNewMatches(res.newMatches);
      return res;
    },
  });

  // Mark notifications as read
  const markNotificationsAsRead = useMutation({
    mutationFn: () => apiRequest("/api/notifications/mark-read", { method: "POST" }),
    onSuccess: () => {
      //console.log("Notifications marked as read successfully.");
      setUnreadMessages(0);
    },
  });

  // Establish WebSocket connection
  useEffect(() => {
    const connectWebSocket = async () => {
      //console.log("Starting WebSocket connection setup...");

      const token = localStorage.getItem("token");
      if (!token) {
        console.error("No token available. WebSocket connection cannot be established.");
        return;
      }

      //console.log("Token retrieved, proceeding to fetch matches...");

      // Fetch all matches of the user
      const matchesResponse = await apiRequest("/api/matches");
      const matchIds = matchesResponse.map(match => match.id); // Assuming each match has an `id`
      //console.log("Match IDs retrieved:", matchIds);

      const newWs: WebSocket[] = []; // Create a new array to store WebSockets
      matchIds.forEach(matchId => {
        //console.log(`Connecting WebSocket for match ID: ${matchId}`);
        const websocket = new WebSocket(`ws://localhost:3000/ws/chat/${matchId}?token=${token}`);

        websocket.onmessage = (event) => {
          const newMessage = JSON.parse(event.data);
        
          // Trigger a refetch of notifications when a new message is received
          queryClient.invalidateQueries({ queryKey: ['notifications'] });
        };

        websocket.onclose = () => {
          //console.log(`WebSocket connection for match ID ${matchId} closed. Retrying in 5 seconds...`);
          setTimeout(connectWebSocket, 5000); // Reconnect after 5 seconds if closed
        };

        newWs.push(websocket); // Add the WebSocket to the array
      });

      setWs(newWs); // Update the state with the new WebSocket array
      setIsConnecting(false); // WebSocket connections are now established
      //console.log("WebSocket connections are now established.");
    };

    connectWebSocket();

    return () => {
      //console.log("Component unmounting. Closing all WebSocket connections...");
      ws.forEach(connection => connection.close());
    };
  }, []); // Empty dependency array to run this effect once

  const handleLogout = () => {
    //console.log("Logging out...");
    localStorage.removeItem("token");
    localStorage.removeItem("user");
    toast({ title: "Logged out successfully", description: "See you next time!" });
    navigate("/");
  };

  const handleNavigateToChats = async () => {
    //console.log("Attempting to navigate to /chats...");

    // Wait until the WebSocket connections are established before navigating
    if (isConnecting) {
      //console.log("WebSocket is still connecting, cannot navigate to chats yet.");
      return;
    }

    //console.log("WebSocket connection established, proceeding to navigate to /chats...");
    //console.log("Marking notifications as read...");
    await markNotificationsAsRead.mutateAsync();
    navigate("/chats");
  };

  return (
    <nav className="bg-white shadow-md p-4">
      <div className="max-w-4xl mx-auto flex justify-between items-center">
        <h1
          onClick={() => {
            //console.log("Navigating to dashboard...");
            navigate("/dashboard");
          }}
          className="text-2xl font-bold bg-gradient-to-r from-match-light to-match-dark text-transparent bg-clip-text cursor-pointer"
        >
          Match Me
        </h1>
        <div className="flex gap-4">
          <Button variant="ghost" className="flex items-center gap-2" onClick={() => {
            //console.log("Navigating to dashboard...");
            navigate("/dashboard");
          }}>
            <Users className="w-4 h-4" />
            Matches
          </Button>
          <Button variant="ghost" className="flex items-center gap-2" onClick={() => {
            //console.log("Navigating to profile...");
            navigate("/profile");
          }}>
            <UserRound className="w-4 h-4" />
            Profile
          </Button>
          <Button variant="ghost" className="flex items-center gap-2 relative" onClick={() => {
            //console.log("Navigating to matches...");
            navigate("/matches");
          }}>
            <Heart className="w-4 h-4" />
            Connections
            {newMatches > 0 && (
              <span className="absolute -top-1 -right-1 bg-red-500 text-white text-xs rounded-full w-5 h-5 flex items-center justify-center">
                {newMatches}
              </span>
            )}
          </Button>
          <Button
            variant="ghost"
            className="flex items-center gap-2 relative"
            onClick={handleNavigateToChats} // Updated to use the new function
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
