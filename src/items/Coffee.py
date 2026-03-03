from src.database.other import reset_user_limit
from src.items.Item import Item, ItemType, ItemRarity


class Coffee(Item):
    def __init__(self):
        super().__init__(
            name="café",
            price=50,
            description="Réveille tes sens. Réinitialise le cooldown du !daily.",
            item_type=ItemType.consumable,
            rarity=ItemRarity.common,
            pet_effet={"stat": "speed", "amount": 5}
        )

    async def use(self, ctx, **kwargs):
        reset_user_limit(ctx.author.id, "daily")
        await ctx.send("☕ **Gloupgloup...** Tu es réveillé ! Ton `!daily` est de nouveau disponible.")
        return True