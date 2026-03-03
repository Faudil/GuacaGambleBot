import random

from src.database.balance import update_balance
from src.items.Item import ItemRarity, ItemType, Item


class FortuneCookie(Item):
    def __init__(self):
        super().__init__(
            name="Fortune Cookie",
            price=20,
            description="🥠 Un biscuit délicieux avec un message prémonitoire.",
            rarity=ItemRarity.common,
            item_type=ItemType.consumable,
            pet_effet={"stat": "crit_c", "amount": 1}
        )

    async def use(self, ctx, **kwargs):
        messages = [
            "Tu vas bientôt devenir riche... ou pas.",
            "Méfie-toi des perroquets.",
            "Le prochain !coinflip sera le bon.",
            "Investis dans le Bitcoin maintenant.",
            "L'amour est au coin de la rue, l'argent aussi.",
            "Tu devrais arrêter de parier (lol)."
        ]
        msg = random.choice(messages)
        update_balance(ctx.author.id, 5)

        await ctx.send(f"🥠 **Crac !** Le papier dit : *\"{msg}\"* (Tu trouves aussi $5).")
        return True