from src.models.NPC import NPC, NPCRegistry
from src.utils.i18n import t

@NPCRegistry.register
class GambleBotNPC(NPC):
    id = "gamblebot"
    name = "GambleBot"
    emoji = "🤖"
    color = 0xffd700 # Gold

    def get_gift_preferences(self):
        return {
            "pépit d'or": 2.0,
            "pépite d'or": 2.0,
            "gold nugget": 2.0,
            "cheat coin": 5.0,
            "jeton de casino": 3.0,
            "casino token": 3.0,
            "rotten plant": 0.1,
            "plante pourrie": 0.1
        }

    def get_bonus(self, user_id: int):
        from src.database.npc import get_reputation
        rep_data = get_reputation(user_id, self.id)
        lvl = rep_data["level"]
        
        # Example: 2% shop discount per level (max 20%)
        discount = min(lvl * 0.02, 0.20)
        return {"shop_discount": discount}

    def get_greeting(self, user_id: int, lang: str):
        from src.database.npc import get_reputation
        rep_data = get_reputation(user_id, self.id)
        lvl = rep_data["level"]
        
        if lvl >= 5:
            return t("npcs.gamblebot.greeting_high", lang, default="Salut, partenaire ! On plume qui aujourd'hui ?")
        elif lvl >= 2:
            return t("npcs.gamblebot.greeting_med", lang, default="Content de te revoir. Tu as misé juste, on dirait.")
        else:
            return t("npcs.gamblebot.greeting_low", lang, default="Bonjour, novice. Tu es là pour dépenser ou pour gagner ?")
