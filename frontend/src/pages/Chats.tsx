import { useEffect, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Card } from "@/components/ui/card";
import { useToast } from "@/hooks/use-toast";
import { Header } from "@/components/Header";
import { UserStatus } from "@/components/UserStatus";
import { apiRequest } from "@/lib/api";
import { Chat } from "@/components/Chat";
import { Avatar, AvatarImage, AvatarFallback } from "@/components/ui/avatar";

interface ChatData {
  matchId: number;
  otherUserId: number;
  otherUserName: string;
  otherUserPicture?: string;
  messages: Message[];
  lastMessage?: string;
  unreadCount: number;
}

interface Message {
  id: string;
  content: string;
  senderId: number;
  timestamp: string;
}

const Chats = () => {
  const navigate = useNavigate();
  const { toast } = useToast();
  const [selectedChat, setSelectedChat] = useState<ChatData | null>(null);
  const [chats, setChats] = useState<ChatData[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const currentUserId = parseInt(localStorage.getItem("user") ? JSON.parse(localStorage.getItem("user") || "{}").id : "0");
  const { matchId } = useParams();

  useEffect(() => {
    const token = localStorage.getItem("token");
    if (!token) {
      navigate("/");
      return;
    }
    fetchChats();
  }, [navigate]);

  useEffect(() => {
    if (matchId && chats.length > 0) {
      const targetChat = chats.find(chat => chat.matchId === parseInt(matchId));
      if (targetChat) {
        setSelectedChat(targetChat);
      } else {
        navigate('/chats');
      }
    }
  }, [matchId, chats, navigate]);

  const fetchChats = async () => {
    try {
      setIsLoading(true);
      const data = await apiRequest('/chat');
  
      if (!Array.isArray(data)) {
        console.error("Expected data to be an array, got:", data);
        setChats([]);
        return;
      }
  
      const chatsWithMessages = await Promise.all(
        data.map(async (chat: any) => {
          if (!chat.match_id) {
            return null;
          }
  
          try {
            const messages = await apiRequest(`/chat/${chat.match_id}/messages`);
            return {
              matchId: chat.match_id,
              otherUserId: chat.other_user_id,
              otherUserName: chat.other_user_name,
              otherUserPicture: chat.other_user_picture,
              messages: messages || [],
              lastMessage: messages && messages.length > 0
                ? messages[messages.length - 1].content
                : undefined,
              unreadCount: chat.unread_count || 0,
            } as ChatData;
          } catch (error) {
            console.error(`Failed to fetch messages for chat ${chat.match_id}:`, error);
            return {
              matchId: chat.match_id,
              otherUserId: chat.other_user_id,
              otherUserName: chat.other_user_name,
              otherUserPicture: chat.other_user_picture,
              messages: [],
              lastMessage: undefined,
              unreadCount: chat.unread_count || 0,
            } as ChatData;
          }
        })
      );
  
      const validChats = chatsWithMessages.filter((chat): chat is ChatData =>
        chat !== null && typeof chat?.matchId === 'number'
      );
  
      setChats(validChats);
    } catch (error) {
      console.error("Error fetching chats:", error);
      toast({
        title: "Error",
        description: "Failed to load chats. Please try again.",
        variant: "destructive",
      });
    } finally {
      setIsLoading(false);
    }
  };

  const handleChatSelect = async (chat: ChatData) => {
    setSelectedChat(chat);
    navigate(`/chats/${chat.matchId}`);
    
    try {
      await apiRequest(`/chat/${chat.matchId}/messages/read`, { method: 'POST' });
      fetchChats();
    } catch (error) {
      console.error("Error marking messages as read:", error);
    }
  };

  if (isLoading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-match-light/10 to-match-dark/10">
        <Header />
        <div className="max-w-6xl mx-auto p-8">
          <div className="text-center">Loading chats...</div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-match-light/10 to-match-dark/10">
      <Header />
      <div className="max-w-6xl mx-auto p-8">
        <div className="bg-white rounded-xl shadow-lg">
          <div className="grid grid-cols-3 min-h-[600px]">
            <div className="border-r">
              <div className="p-4 border-b">
                <h2 className="text-xl font-semibold">Chats</h2>
              </div>
              <ScrollArea className="h-[calc(600px-4rem)]">
                {chats.length === 0 ? (
                  <div className="p-4 text-center text-gray-500">
                    You don't have any chats yet. Start matching with people to
                    begin chatting!
                  </div>
                ) : (
                  chats.map((chat) => (
                    <Card
                      key={chat.matchId}
                      className={`m-2 p-4 cursor-pointer transition-colors hover:bg-gray-50 ${
                        selectedChat?.matchId === chat.matchId
                          ? "bg-gray-100"
                          : ""
                      }`}
                      onClick={() => handleChatSelect(chat)}
                    >
                      <div className="flex items-center gap-2">
                        <Avatar className="h-8 w-8">
                          <AvatarImage src={chat.otherUserPicture} alt={chat.otherUserName} />
                          <AvatarFallback>{chat.otherUserName[0]}</AvatarFallback>
                        </Avatar>
                        <div className="flex-1">
                          <div className="flex items-center gap-2">
                            <h3 className="font-medium">{chat.otherUserName}</h3>
                            <UserStatus userId={chat.otherUserId} />
                            {chat.unreadCount > 0 && (
                              <span className="bg-red-500 text-white text-xs rounded-full w-5 h-5 flex items-center justify-center">
                                {chat.unreadCount}
                              </span>
                            )}
                          </div>
                          {chat.lastMessage && (
                            <p className="text-sm text-gray-500 truncate">
                              {chat.lastMessage}
                            </p>
                          )}
                        </div>
                      </div>
                    </Card>
                  ))
                )}
              </ScrollArea>
            </div>

            <div className="col-span-2 flex flex-col">
              {selectedChat ? (
                <Chat
                  matchId={selectedChat.matchId}
                  currentUserId={currentUserId}
                  otherUserName={selectedChat.otherUserName}
                  otherUserPicture={selectedChat.otherUserPicture}
                  otherUserId={selectedChat.otherUserId}
                />
              ) : (
                <div className="flex-1 flex items-center justify-center text-gray-500">
                  Select a chat to start messaging
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Chats;
