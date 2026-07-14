from src.models.NPC import NPC, NPCRegistry
from src.database.npc import get_reputation
from src.utils.i18n import t
from src.items.Hook import Hook
from src.items.FishingLoot import Trout, Salmon, Swordfish

@NPCRegistry.register
class IrianNPC(NPC):
    id = "irian"
    name = "Irian"
    emoji = "🎣"
    color = 0x4682b4 # Steel Blue

    def get_gift_preferences(self):
        return {
            "tentacule de kraken": 2.0,
            "baleine": 2.0,
            "requin": 2.0,
            "truite": 1.5,
            "saumon": 1.5,
            "hameçon": 1.5,
            "sardine": 1.5,
            "carpe": 1.5,
            "poisson-globe": 1.5,
            "espadon": 1.5,
            "pièce truquée": 0.5
        }

    def get_bonus(self, user_id: int):
        rep_data = get_reputation(user_id, self.id)
        lvl = rep_data["level"]
        # Increase fishing bite window by level * 0.1s
        return {"fishing_time_bonus": lvl * 0.1}

    def get_shop_inventory(self, current_level: int) -> list:
        items = []
        if current_level >= 1:
            items.append(Hook())
        if current_level >= 2:
            items.append(Trout())
        if current_level >= 3:
            items.append(Salmon())
        if current_level >= 4:
            items.append(Swordfish())
        return items

    def get_greeting(self, user_id: int, lang: str):
        rep_data = get_reputation(user_id, self.id)
        lvl = rep_data["level"]
        if lvl >= 5:
            return t("npcs.irian.greeting_high", lang, default="Ah, mon capitaine ! Les marées nous sont favorables aujourd'hui.")
        elif lvl >= 2:
            return t("npcs.irian.greeting_med", lang, default="Salut marin. Tu as senti l'odeur du grand large récemment ?")
        else:
            return t("npcs.irian.greeting_low", lang, default="Chut... Tu vas faire fuir les poissons.")
