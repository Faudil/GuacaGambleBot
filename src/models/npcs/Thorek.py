from src.models.NPC import NPC, NPCRegistry
from src.database.npc import get_reputation
from src.utils.i18n import t
from src.items.Magnet import Magnet
from src.items.MiningLoot import IronOre, SilverOre, GoldNugget

@NPCRegistry.register
class ThorekNPC(NPC):
    id = "thorek"
    name = "Thorek"
    emoji = "⚒️"
    color = 0x8b4513 # Saddle Brown

    def get_gift_preferences(self):
        return {
            "pépite d'or": 2.0,
            "diamant brut": 2.0,
            "platine": 2.0,
            "minerai de fer": 1.5,
            "minerai de cuivre": 1.5,
            "aimant": 1.5,
            "plante pourrie": 0.5
        }

    def get_bonus(self, user_id: int):
        rep_data = get_reputation(user_id, self.id)
        lvl = rep_data["level"]
        # Reduce mining risk by level * 2
        return {"mining_risk_reduction": lvl * 2}

    def get_shop_inventory(self, current_level: int) -> list:
        items = []
        if current_level >= 1:
            items.append(Magnet())
        if current_level >= 2:
            items.append(IronOre())
        if current_level >= 3:
            items.append(SilverOre())
        if current_level >= 4:
            items.append(GoldNugget())
        return items

    def get_greeting(self, user_id: int, lang: str):
        rep_data = get_reputation(user_id, self.id)
        lvl = rep_data["level"]
        if lvl >= 5:
            return t("npcs.thorek.greeting_high", lang, default="Salut l'ami ! Ma forge est toujours ouverte pour un bon mineur comme toi.")
        elif lvl >= 2:
            return t("npcs.thorek.greeting_med", lang, default="Ah, te voilà ! Tu as trouvé du bon minerai récemment ?")
        else:
            return t("npcs.thorek.greeting_low", lang, default="Qu'est-ce que tu veux ? Si tu n'as pas de pioche, tu perds ton temps.")
