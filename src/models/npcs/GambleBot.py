from src.items.CasinoToken import CasinoToken
from src.items.CheatCoin import CheatCoin
from src.items.MiningLoot import GoldNugget
from src.items.VipTicket import VipTicket
from src.models.NPC import NPC, NPCRegistry
from src.database.npc import get_reputation
from src.utils.i18n import t

@NPCRegistry.register
class GambleBotNPC(NPC):
    id = "gamblebot"
    name = "GambleBot"
    emoji = "🤖"
    color = 0xffd700 # Gold

    def get_gift_preferences(self):
        return {
            GoldNugget().name: 2.0,
            CheatCoin().name: 5.0,
            VipTicket().name: 3.0,
        }

    def get_bonus(self, user_id: int):
        rep_data = get_reputation(user_id, self.id)
        lvl = rep_data["level"]
        
        # Example: 2% shop discount per level (max 20%)
        discount = min(lvl * 0.02, 0.20)
        return {"shop_discount": discount}

    def get_shop_inventory(self, current_level: int) -> list:
        items = []
        if current_level >= 1:
            items.append(CasinoToken())
        if current_level >= 2:
            items.append(CheatCoin())
        if current_level >= 3:
            items.append(VipTicket())
        return items

    def get_greeting(self, user_id: int, lang: str):
        rep_data = get_reputation(user_id, self.id)
        lvl = rep_data["level"]
        
        if lvl >= 5:
            return t("npcs.gamblebot.greeting_high", lang, default="Salut, partenaire ! On plume qui aujourd'hui ?")
        elif lvl >= 2:
            return t("npcs.gamblebot.greeting_med", lang, default="Content de te revoir. Tu as misé juste, on dirait.")
        else:
            return t("npcs.gamblebot.greeting_low", lang, default="Bonjour humain. Avez vous tenté votre chance aujourd'hui ?")
