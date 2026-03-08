import asyncio
import discord
from src.models.Pet import Pet

async def simulate_battle(
    pet1: Pet,
    pet2: Pet,
    msg: discord.Message = None,
    embed: discord.Embed = None,
    update_embed_func = None,
    sleep_time: float = 0.5,
    send_messages: bool = True,
    log_size: int = 10,
    journal_title: str = "📜 **Journal de combat :**\n\n"
):
    """
    Simulate a battle between two pets.

    :param pet1: The first pet.
    :param pet2: The second pet.
    :param msg: The discord message to update (optional).
    :param embed: The discord embed to update (optional).
    :param update_embed_func: A function to call to update the embed fields (optional).
    :param sleep_time: Time to sleep between each attack animation.
    :param send_messages: Whether to send discord message updates.
    :param log_size: Maximum number of lines in the combat log.
    :param journal_title: Title of the combat log.
    :return: None
    """
    log = []
    actions_count = 0
    atb_pet1 = 0
    atb_pet2 = 0

    while pet1.is_alive and pet2.is_alive and actions_count <= 70:
        atb_pet1 += pet1.real_speed
        atb_pet2 += pet2.real_speed

        while (atb_pet1 >= 100 or atb_pet2 >= 100) and pet1.is_alive and pet2.is_alive:
            if atb_pet1 >= 100 and atb_pet2 >= 100:
                if atb_pet1 > atb_pet2:
                    attacker, defender = pet1, pet2
                    atb_pet1 -= 100
                    atb_pet2 += pet1.real_speed
                elif atb_pet2 > atb_pet1:
                    attacker, defender = pet2, pet1
                    atb_pet2 -= 100
                    atb_pet1 += pet2.real_speed

                else: 
                    if pet1.real_speed >= pet2.real_speed:
                        attacker, defender = pet1, pet2
                        atb_pet1 -= 100
                    else:
                        attacker, defender = pet2, pet1
                        atb_pet2 -= 100
            elif atb_pet1 >= 100:
                attacker, defender = pet1, pet2
                atb_pet1 -= 100
            else:
                attacker, defender = pet2, pet1
                atb_pet2 -= 100

            fatigue_mult = 1.0
            if actions_count > 25:
                fatigue_mult = max(0.2, 1.0 - ((actions_count - 50) * 0.05))
                
            action_text = attacker.attack(defender, fatigue_mult=fatigue_mult)
            log.append(action_text)

            if len(log) > log_size:
                log.pop(0)

            if send_messages and msg and embed and update_embed_func:
                update_embed_func()
                embed.description = f"{journal_title}" + "\n".join(log)
                await msg.edit(embed=embed)
                await asyncio.sleep(sleep_time)

            actions_count += 1
