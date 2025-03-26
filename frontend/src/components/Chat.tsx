import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "./ui/button";
import { ScrollArea } from "./ui/scroll-area";
import { Card } from "./ui/card";
import { Textarea } from "./ui/textarea";
import { useState, useEffect, useRef } from "react";
import { Link } from "react-router-dom";
import { useToast } from "./ui/use-toast";
import { Avatar, AvatarImage, AvatarFallback } from "./ui/avatar";
import { apiRequest } from "@/lib/api";
import { UserStatus } from "./UserStatus";

interface Message {
  id: string;
  sender_id: number;
  content: string;
  timestamp: string;
  read: boolean;
}

interface ChatProps {
  matchId?: number;
  currentUserId?: number;
  otherUserName?: string;
  otherUserPicture?: string;
  otherUserId?: number;
}

export const Chat = ({ matchId, currentUserId, otherUserName, otherUserPicture, otherUserId }: ChatProps) => {
  const queryClient = useQueryClient();
  const [message, setMessage] = useState("");
  const { toast } = useToast();
  const [localMessages, setLocalMessages] = useState<Message[]>([]);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout>();
  const maxRetries = 5;
  const [retryCount, setRetryCount] = useState(0);
  const [isTyping, setIsTyping] = useState(false);
  const typingTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  const { data: initialMessages, refetch } = useQuery({
    queryKey: ['messages', matchId],
    queryFn: async () => {
      if (!matchId) return { messages: [] };
      const response = await apiRequest(`/chat/${matchId}/messages`);
      return response;
    },
    enabled: !!matchId,
  });

  useEffect(() => {
    if (initialMessages?.messages) {
      // Reverse the order of messages to show oldest first
      setLocalMessages(initialMessages.messages);
    }
  }, [initialMessages]);

  const connectWebSocket = () => {
    if (!matchId || !currentUserId) {
      console.log('No match ID or user ID provided');
      return;
    }

    if (wsRef.current?.readyState === WebSocket.OPEN) {
      console.log('WebSocket already connected');
      return;
    }

    const token = localStorage.getItem('token');
    if (!token) {
      console.error('No authentication token found');
      return;
    }

    console.log('Connecting WebSocket for match:', matchId);
    const wsUrl = `${import.meta.env.VITE_WS_URL}/ws/chat/${matchId}?token=Bearer ${token}`;
    const websocket = new WebSocket(wsUrl);
    
    websocket.onopen = () => {
      console.log('WebSocket Connected for match:', matchId);
      setRetryCount(0);
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
    };

    websocket.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.typing !== undefined) {
        setIsTyping(data.typing && data.user_id !== currentUserId);
        return;
      }
      const newMessage = data;
      console.log('Received message:', newMessage);
      setLocalMessages(prev => {
        if (!Array.isArray(prev)) {
          return [newMessage];
        }
        if (prev.some(msg => msg.id === newMessage.id)) {
          return prev;
        }
        return [...prev, newMessage];
      });

      // Trigger a refetch of notifications when a new message is received
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
    };

    websocket.onclose = (event) => {
      console.log("WebSocket connection closed for match:", matchId);
      wsRef.current = null;

      if (retryCount < maxRetries) {
        const timeout = Math.min(1000 * Math.pow(2, retryCount), 10000);
        console.log(`Attempting to reconnect in ${timeout}ms... (Attempt ${retryCount + 1}/${maxRetries})`);
        reconnectTimeoutRef.current = setTimeout(() => {
          setRetryCount(prev => prev + 1);
          connectWebSocket();
        }, timeout);
      } else {
        toast({
          title: "Connection Lost",
          description: "Unable to maintain chat connection. Please refresh the page.",
          variant: "destructive",
        });
      }
    };

    websocket.onerror = (error) => {
      console.error('WebSocket error for match:', matchId, error);
      toast({
        title: "Connection Error",
        description: "There was an error with the chat connection",
        variant: "destructive",
      });
    };

    wsRef.current = websocket;
  };

  useEffect(() => {
    if (matchId && currentUserId) {
      console.log('Initializing WebSocket connection for match:', matchId);
      connectWebSocket();
    }

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.close(1000, "Component unmounting");
      }
    };
  }, [matchId, currentUserId]);

  useEffect(() => {
    if (messagesEndRef.current) {
      messagesEndRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [localMessages]);

  useEffect(() => {
    if (matchId && currentUserId) {
      apiRequest(`/chat/${matchId}/messages/read`, { method: 'POST' }).then(() => {
        refetch();
      });
    }
  }, [matchId, currentUserId, refetch]);

  const sendMessage = async () => {
    if (!message.trim() || !matchId || !currentUserId) {
      if (!matchId || !currentUserId) {
        toast({
          title: "Error",
          description: "Cannot send message - chat not properly initialized",
          variant: "destructive",
        });
      }
      return;
    }
    
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      console.log('WebSocket not ready, attempting to reconnect...');
      connectWebSocket();
      toast({
        title: "Connection Error",
        description: "Reconnecting to chat...",
        variant: "destructive",
      });
      return;
    }

    try {
      const messageData = {
        id: `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
        content: message,
        sender_id: currentUserId,
        timestamp: new Date().toISOString(),
        read: false
      };
      
      console.log('Sending message:', messageData);
      wsRef.current.send(JSON.stringify({ ...messageData, match_id: matchId }));
      
      setLocalMessages(prev => [...(Array.isArray(prev) ? prev : []), messageData]);
      setMessage("");
      
      // Refetch messages after sending to ensure consistency
      setTimeout(() => refetch(), 500);
    } catch (error) {
      console.error('Error sending message:', error);
      toast({
        title: "Error",
        description: "Failed to send message",
        variant: "destructive",
      });
    }
  };

  const handleTyping = () => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;
    wsRef.current.send(JSON.stringify({ typing: true, match_id: matchId, user_id: currentUserId }));
    if (typingTimeoutRef.current) clearTimeout(typingTimeoutRef.current);
    typingTimeoutRef.current = setTimeout(() => {
      wsRef.current?.send(JSON.stringify({ typing: false, match_id: matchId, user_id: currentUserId }));
    }, 1000);
  };

  if (!matchId || !currentUserId) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-500">
        Select a chat to start messaging
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      <div className="p-4 border-b">
        <div className="flex items-center gap-2">
          <Avatar className="h-8 w-8">
            <AvatarImage src={otherUserPicture} alt={otherUserName} />
            <AvatarFallback>{otherUserName?.[0]}</AvatarFallback>
          </Avatar>
          <div>
            <h3 className="font-medium">{otherUserName}</h3>
            <UserStatus userId={otherUserId} />
          </div>
        </div>
      </div>

      <ScrollArea className="flex-1 p-4">
        <div className="space-y-4">
          {localMessages.map((msg) => (
            <div
              key={msg.id}
              className={`flex ${
                msg.sender_id === currentUserId ? "justify-end" : "justify-start"
              }`}
            >
              <div
                className={`max-w-[70%] rounded-lg p-3 ${
                  msg.sender_id === currentUserId
                    ? "bg-primary text-primary-foreground"
                    : "bg-muted"
                }`}
              >
                <p className="text-sm">{msg.content}</p>
                <p className="text-xs opacity-70 mt-1">
                  {new Date(msg.timestamp).toLocaleTimeString()}
                </p>
              </div>
            </div>
          ))}
          <div ref={messagesEndRef} />
        </div>
      </ScrollArea>

      <div className="p-4 border-t">
        <div className="flex gap-2">
          <Textarea
            value={message}
            onChange={(e) => {
              setMessage(e.target.value);
              handleTyping();
            }}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                sendMessage();
              }
            }}
            placeholder="Type a message..."
            className="min-h-[60px]"
          />
          <Button onClick={sendMessage} className="self-end">
            Send
          </Button>
        </div>
      </div>
    </div>
  );
};
