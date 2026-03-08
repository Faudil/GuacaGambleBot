from src.database.other import reset_user_limit
from src.items.Item import Item, ItemType, ItemRarity


class Bow(Item):
    def __init__(self):
        super().__init__(
            name="Arc",
            price=300,
            description="Aide à la chasse ! Réinitialise le cooldown de !hunt.",
            item_type=ItemType.consumable,
            rarity=ItemRarity.common,
            pet_effet={"stat": "atk", "amount": 3}
        )

    async def use(self, ctx, **kwargs):
        reset_user_limit(ctx.author.id, "hunt")
        await ctx.send("☕ **Ploc !** Ton `!hunt` est de nouveau disponible.")
        return True
